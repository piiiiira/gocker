package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"go.yaml.in/yaml/v4"
)

type Image struct {
	Namespace string `yaml:"namespace"`
	Image     string `yaml:"image"`
	Registry  string `yaml:"registry"`
}

type Config struct {
	Tag         string  `yaml:"tag"`
	GithubToken string  `yaml:"github_token"`
	Images      []Image `yaml:"images"`
}

type TagDigest struct {
	Name       string
	Digest     string
	LastPushed string
}

func getTagDigest(tag, token, namespace, image, registry string) (string, string, error) {
	var url string

	if registry == "ghcr" {
		url = fmt.Sprintf("https://ghcr.io/v2/%s/%s/manifests/%s", namespace, image, tag)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		res, _ := http.DefaultClient.Do(req)
		defer res.Body.Close()

		if res.StatusCode == 200 {
			digest := res.Header.Get("Docker-Content-Digest")
			if digest != "" {
				return digest, time.Now().Format(time.RFC3339), nil
			}
		}
		return "", "", fmt.Errorf("digest not found")
	}

	// Default to Docker Hub
	url = fmt.Sprintf("https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags/%s", namespace, image, tag)
	req, _ := http.NewRequest("GET", url, nil)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var result map[string]any
	json.Unmarshal(body, &result)

	if result["digest"] != nil {
		if result["tag_last_pushed"] != nil {
			lastPushed := result["tag_last_pushed"].(string)
			return result["digest"].(string), lastPushed, nil
		}
	}
	return "", "", fmt.Errorf("digest not found")
}

func isSemver(version string) bool {
	pattern := `^v?\d+(\.\d+)?(\.\d+)?$`
	matched, _ := regexp.MatchString(pattern, version)
	return matched
}

func getAllTags(token, namespace, image, registry string) ([]TagDigest, error) {
	switch registry {
	case "ghcr":
		url := fmt.Sprintf("https://ghcr.io/v2/%s/%s/tags/list", namespace, image)
		tokenEncoded := fmt.Sprintf("Bearer %s", base64.StdEncoding.EncodeToString([]byte(token)))

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Add("Authorization", tokenEncoded)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, body)
		}

		var result map[string]any
		json.Unmarshal(body, &result)

		var tags []TagDigest
		if tagNames, ok := result["tags"].([]any); ok {
			for _, tagName := range tagNames {
				name := tagName.(string)

				// Fetch manifest for each tag to get digest and metadata
				manifestURL := fmt.Sprintf("https://ghcr.io/v2/%s/%s/manifests/%s", namespace, image, name)
				manifestReq, err := http.NewRequest("GET", manifestURL, nil)
				if err != nil {
					continue
				}

				manifestReq.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
				manifestReq.Header.Add("Authorization", tokenEncoded)

				manifestResp, err := http.DefaultClient.Do(manifestReq)
				if err != nil {
					continue
				}
				defer manifestResp.Body.Close()

				digest := ""
				lastPushed := ""

				if manifestResp.StatusCode == 200 {
					digest = manifestResp.Header.Get("Docker-Content-Digest")

					// Try to get Last-Modified header for last pushed date
					if lastModified := manifestResp.Header.Get("Last-Modified"); lastModified != "" {
						lastPushed = lastModified
					}
				}

				tags = append(tags, TagDigest{Name: name, Digest: digest, LastPushed: lastPushed})
			}
		}

		return tags, nil

	case "docker":
		url := fmt.Sprintf("https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags?page_size=100", namespace, image)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, body)
		}

		var resultAll map[string]any
		json.Unmarshal(body, &resultAll)

		var tags []TagDigest
		if results, ok := resultAll["results"].([]any); ok {
			for _, tag := range results {
				tagMap := tag.(map[string]any)

				var name string
				if tagMap["name"] != nil {
					name = tagMap["name"].(string)
				}

				var digest string
				if tagMap["digest"] != nil {
					digest = tagMap["digest"].(string)
				}

				var lastPushed string
				if tagMap["tag_last_pushed"] != nil {
					lastPushed = tagMap["tag_last_pushed"].(string)
				}
				tags = append(tags, TagDigest{Name: name, Digest: digest, LastPushed: lastPushed})
			}
		}

		return tags, nil

	default:
		return []TagDigest{}, fmt.Errorf("unsupported registry: %s", registry)
	}
}

func getAllTagsGHCR(token, namespace, image string) ([]TagDigest, error) {
	url := fmt.Sprintf("https://ghcr.io/v2/%s/%s/tags/list", namespace, image)
	tokenEncoded := fmt.Sprintf("Bearer %s", base64.StdEncoding.EncodeToString([]byte(token)))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", tokenEncoded)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, body)
	}

	var result map[string]any
	json.Unmarshal(body, &result)

	var tags []TagDigest
	if tagNames, ok := result["tags"].([]any); ok {
		for _, tagName := range tagNames {
			name := tagName.(string)
			tags = append(tags, TagDigest{Name: name, Digest: "", LastPushed: ""})
		}
	}

	return tags, nil
}

func getConfig(configFile string) (Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func main() {
	config, err := getConfig("config.yml")
	if err != nil {
		log.Fatal("Error getting config:", err)
	}

	fmt.Println("Tag:", config.Tag)

	for _, img := range config.Images {
		fmt.Printf("\n*** %s/%s (%s) ***\n", img.Namespace, img.Image, img.Registry)

		digest, lastPushed, err := getTagDigest(config.Tag, config.GithubToken, img.Namespace, img.Image, img.Registry)
		if err != nil {
			fmt.Println("Error getting latest digest:", err)
			continue
		}

		lastPushedFormated, _ := time.Parse(time.RFC3339, lastPushed)
		daysSince := int(time.Since(lastPushedFormated).Hours() / 24)
		fmt.Println("Digest:", digest)
		fmt.Println("Pushed:", daysSince, "days ago")

		tags, err := getAllTags(config.GithubToken, img.Namespace, img.Image, img.Registry)
		if err != nil {
			fmt.Println("Error getting tags:", err)
			continue
		}

		for _, tag := range tags {
			if tag.Digest == digest && isSemver(tag.Name) {
				fmt.Println(tag.Name)
			}
		}
	}
}
