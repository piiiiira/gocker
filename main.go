package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TagDigest struct {
	Name       string
	Digest     string
	LastPushed string
}

func getLatestDigest(namespace, image string) (string, error) {
	url := fmt.Sprintf("https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags/latest", namespace, image)

	req, _ := http.NewRequest("GET", url, nil)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var result map[string]any
	json.Unmarshal(body, &result)

	if digest, ok := result["digest"].(string); ok {
		return digest, nil
	}
	return "", fmt.Errorf("digest not found")
}

func getAllTags(namespace, image string) ([]TagDigest, error) {
	url := fmt.Sprintf("https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags?page_size=100", namespace, image)

	req, _ := http.NewRequest("GET", url, nil)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var resultAll map[string]any
	json.Unmarshal(body, &resultAll)

	var tags []TagDigest
	if results, ok := resultAll["results"].([]any); ok {
		for _, tag := range results {
			tagMap := tag.(map[string]any)
			name := tagMap["name"].(string)
			digest := tagMap["digest"].(string)
			lastPushed := tagMap["tag_last_pushed"].(string)
			tags = append(tags, TagDigest{Name: name, Digest: digest, LastPushed: lastPushed})
		}
	}
	return tags, nil
}

func main() {
	namespace := flag.String("namespace", "library", "Docker namespace")
	image := flag.String("image", "nginx", "Docker image name")
	flag.Parse()

	digest, _ := getLatestDigest(*namespace, *image)
	fmt.Println("Latest digest:", digest)

	tags, _ := getAllTags(*namespace, *image)
	for _, tag := range tags {
		if tag.Digest == digest {
			lastPushed, _ := time.Parse(time.RFC3339, tag.LastPushed)
			daysSince := int(time.Since(lastPushed).Hours() / 24)
			fmt.Printf("%s\t%d days ago\n", tag.Name, daysSince)
		}
	}
}
