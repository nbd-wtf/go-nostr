package nip92

import (
	"strconv"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type IMeta []IMetaEntry

func (imeta IMeta) Get(url string) (IMetaEntry, bool) {
	for _, entry := range imeta {
		if entry.URL == url {
			return entry, true
		}
	}
	return IMetaEntry{}, false
}

type IMetaEntry struct {
	URL      string
	Blurhash string
	Width    int
	Height   int
	Alt      string
}

func ParseTags(tags nostr.Tags) IMeta {
	var imeta IMeta
	for i, tag := range tags {
		if len(tag) > 2 && tag[0] == "imeta" {
			entry := IMetaEntry{}
			for _, item := range tag[1:] {
				div := strings.Index(item, " ")
				if div == -1 {
					continue
				}

				switch item[0:div] {
				case "url":
					entry.URL = item[div+1:]
				case "alt":
					entry.Alt = item[div+1:]
				case "blurhash":
					entry.Blurhash = item[div+1:]
				case "dim":
					xySplit := strings.Index(item[div+1:], "x")
					if xySplit == -1 {
						// if any tag is wrong them we don't trust this guy anyway
						return nil
					}

					x, err := strconv.Atoi(item[div+1 : div+1+xySplit])
					if err != nil {
						return nil
					}
					entry.Width = x

					y, err := strconv.Atoi(item[div+1+xySplit+1:])
					if err != nil {
						return nil
					}
					entry.Height = y
				}
			}

			if imeta == nil {
				imeta = make(IMeta, 1, len(tags)-i)
				imeta[0] = entry
			} else {
				imeta = append(imeta, entry)
			}
		}
	}
	return imeta
}
