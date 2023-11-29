# RSS feeds for Automat

## Build Docker image

```bash
docker build . -t automat/mnk-rss
```

## Run server

```bash
docker run -it -p 8010:8010 -v ./config.yaml:/mnk-rss/config.yaml:ro automat/mnk-rss
```

