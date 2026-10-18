package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"go.yaml.in/yaml/v4"
)

type Image struct {
	Namespace string `yaml:"namespace"`
	Image     string `yaml:"image"`
}

type Config struct {
	Images []Image `yaml:"images"`
	Tag    string  `yaml:"tag"`
}

type TagDigest struct {
	Name       string
	Digest     string
	LastPushed string
}

func getTagDigest(tag, namespace, image string) (string, string, error) {
	url := fmt.Sprintf("https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags/%s", namespace, image, tag)

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

			var name string
			if tagMap["name"] != nil {
				name = tagMap["name"].(string)
			} else {
				name = "prout"
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
		fmt.Println("Error getting config:", err)
	}

	fmt.Println("Tag:", config.Tag)

	for _, img := range config.Images {
		fmt.Printf("\n*** %s/%s ***\n", img.Namespace, img.Image)

		digest, lastPushed, err := getTagDigest(config.Tag, img.Namespace, img.Image)
		if err != nil {
			fmt.Println("Error getting latest digest:", err)
			continue
		}

		lastPushedFormated, _ := time.Parse(time.RFC3339, lastPushed)
		daysSince := int(time.Since(lastPushedFormated).Hours() / 24)
		fmt.Println("Digest:", digest)
		fmt.Println("Pushed:", daysSince, "days ago")

		tags, err := getAllTags(img.Namespace, img.Image)
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
