# Gocker

A tool to check Docker image tag versions across multiple repositories to make it easy to update digests.

Gocker retrieves Docker image information from Docker Hub and displays tags matching the specified reference tag's digest, filtering to only semver formats (`1.0.0`, `v2.1`, ...).

Works only with Docker Hub.

## Configuration

`config.yml`

```yaml
tag: latest
images:
  - namespace: library
    image: traefik
  - namespace: library
    image: postgres
```

## Usage

```bash
gocker
```

## Output

```
library/traefik:{TAG}@sha256:0f6150202bc1fed72a7bd2107460eaa510e5b8a905f3fdb5b84f2c8619c33368
v3.7.15
v3.7
3.7.15
3.7
Pushed: 2 days ago

library/postgres:{TAG}@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336
18.6
Pushed: 11 days ago
```

For each configured image, gocker displays:
- Digest of the tag specified in the configuration
- Days since the image was last pushed
- List of tags matching semantic versioning format
