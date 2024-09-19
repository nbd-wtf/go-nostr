package nip94

import (
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

func ParseFileMetadata(event nostr.Event) FileMetadata {
	fm := FileMetadata{}
	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "url":
			fm.URL = tag[1]
		case "x":
			fm.X = tag[1]
		case "ox":
			fm.OX = tag[1]
		case "size":
			fm.Size = tag[1]
		case "dim":
			fm.Dim = tag[1]
		case "magnet":
			fm.Magnet = tag[1]
		case "i":
			fm.TorrentInfoHash = tag[1]
		case "blurhash":
			fm.Blurhash = tag[1]
		case "thumb":
			fm.Image = tag[1]
		case "summary":
			fm.Summary = tag[1]
		}
	}
	return fm
}

type FileMetadata struct {
	Magnet          string
	Dim             string
	Size            string
	Summary         string
	Image           string
	URL             string
	M               string
	X               string
	OX              string
	TorrentInfoHash string
	Blurhash        string
	Thumb           string
	Content         string
}

func (fm FileMetadata) IsVideo() bool { return strings.Split(fm.M, "/")[0] == "video" }
func (fm FileMetadata) IsImage() bool { return strings.Split(fm.M, "/")[0] == "image" }
func (fm FileMetadata) DisplayImage() string {
	if fm.Image != "" {
		return fm.Image
	} else if fm.IsImage() {
		return fm.URL
	} else {
		return ""
	}
}

func (fm FileMetadata) ToTags() nostr.Tags {
	tags := make(nostr.Tags, 0, 12)
	if fm.URL != "" {
		tags = append(tags, nostr.Tag{"url", fm.URL})
	}
	if fm.M != "" {
		tags = append(tags, nostr.Tag{"m", fm.M})
	}
	if fm.X != "" {
		tags = append(tags, nostr.Tag{"x", fm.X})
	}
	if fm.OX != "" {
		tags = append(tags, nostr.Tag{"ox", fm.OX})
	}
	if fm.Size != "" {
		tags = append(tags, nostr.Tag{"size", fm.Size})
	}
	if fm.Dim != "" {
		tags = append(tags, nostr.Tag{"dim", fm.Dim})
	}
	if fm.Magnet != "" {
		tags = append(tags, nostr.Tag{"magnet", fm.Magnet})
	}
	if fm.TorrentInfoHash != "" {
		tags = append(tags, nostr.Tag{"i", fm.TorrentInfoHash})
	}
	if fm.Blurhash != "" {
		tags = append(tags, nostr.Tag{"blurhash", fm.Blurhash})
	}
	if fm.Thumb != "" {
		tags = append(tags, nostr.Tag{"thumb", fm.Thumb})
	}
	if fm.Image != "" {
		tags = append(tags, nostr.Tag{"image", fm.Image})
	}
	if fm.Summary != "" {
		tags = append(tags, nostr.Tag{"summary", fm.Summary})
	}
	return tags
}
