package nip96

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/nbd-wtf/go-nostr"
)

// Upload uploads a file to the provided req.Host.
func Upload(ctx context.Context, req UploadRequest) (*UploadResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	client := http.DefaultClient
	if req.HTTPClient != nil {
		client = req.HTTPClient
	}

	var requestBody bytes.Buffer
	fileHash := sha256.New()
	writer := multipart.NewWriter(&requestBody)
	{
		// Add the file
		fileWriter, err := writer.CreateFormFile("file", req.Filename)
		if err != nil {
			return nil, fmt.Errorf("multipartWriter.CreateFormFile: %w", err)
		}
		if _, err := io.Copy(fileWriter, io.TeeReader(req.File, fileHash)); err != nil {
			return nil, fmt.Errorf("io.Copy: %w", err)
		}

		// Add the other fields
		writer.WriteField("caption", req.Caption)
		writer.WriteField("alt", req.Alt)
		writer.WriteField("media_type", req.MediaType)
		writer.WriteField("content_type", req.ContentType)
		writer.WriteField("no_transform", fmt.Sprintf("%t", req.NoTransform))
		if req.Expiration == 0 {
			writer.WriteField("expiration", "")
		} else {
			writer.WriteField("expiration", strconv.FormatInt(int64(req.Expiration), 10))
		}

		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("multipartWriter.Close: %w", err)
		}
	}

	uploadReq, err := http.NewRequest("POST", req.Host, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	if req.SK != "" {
		if !req.SignPayload {
			fileHash = nil
		}
		auth, err := generateAuthHeader(req.SK, req.Host, fileHash)
		if err != nil {
			return nil, fmt.Errorf("generateAuthHeader: %w", err)
		}
		uploadReq.Header.Set("Authorization", auth)
	}

	resp, err := client.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("httpclient.Do: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusRequestEntityTooLarge:
		return nil, fmt.Errorf("File is too large")

	case http.StatusBadRequest:
		return nil, fmt.Errorf("Bad request")

	case http.StatusForbidden:
		return nil, fmt.Errorf("Unauthorized")

	case http.StatusPaymentRequired:
		return nil, fmt.Errorf("Payment required")

	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		var uploadResp UploadResponse
		if err := jsoniter.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
			return nil, fmt.Errorf("Error decoding JSON: %w", err)
		}
		return &uploadResp, nil

	default:
		return nil, fmt.Errorf("Unexpected error %v", resp.Status)
	}
}

func generateAuthHeader(sk, host string, fileHash hash.Hash) (string, error) {
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		return "", fmt.Errorf("nostr.GetPublicKey: %w", err)
	}

	event := nostr.Event{
		Kind:      27235,
		PubKey:    pk,
		CreatedAt: nostr.Now(),
		Tags: nostr.Tags{
			nostr.Tag{"u", host},
			nostr.Tag{"method", "POST"},
		},
	}
	if fileHash != nil {
		event.Tags = append(event.Tags, nostr.Tag{"payload", hex.EncodeToString(fileHash.Sum(nil))})
	}
	event.Sign(sk)

	b, err := jsoniter.ConfigFastest.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}

	payload := base64.StdEncoding.EncodeToString(b)

	return fmt.Sprintf("Nostr %s", payload), nil
}
