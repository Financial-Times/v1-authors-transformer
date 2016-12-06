# V1 Authors Transformer
[![CircleCI](https://circleci.com/gh/Financial-Times/v1-authors-transformer.svg?style=svg)](https://circleci.com/gh/Financial-Times/v1-authors-transformer) [![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/v1-authors-transformer)](https://goreportcard.com/report/github.com/Financial-Times/v1-authors-transformer) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/v1-authors-transformer/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/v1-authors-transformer?branch=master) [![codecov](https://codecov.io/gh/Financial-Times/v1-authors-transformer/branch/master/graph/badge.svg)](https://codecov.io/gh/Financial-Times/v1-authors-transformer)

An API for pulling in and transforming V1/TME Authors into the UPP representation of an Author 

## Installation

For the first time:

`go get github.com/Financial-Times/v1-authors-transformer`

or update:

`go get -u github.com/Financial-Times/v1-authors-transformer`

## Running

`$GOPATH/bin/v1-authors-transformer --bertha-service-url={bertha url} --tme-username={tme-username} --port={port} --tme-password={tme-password} --token={tme-token} --base-url={base-url for the app} --tme-base-url={tme-base-url} --maxRecords={maxRecords} --batchSize={batchSize} --cache-file-name={cache-file-name}`

TME credentials are mandatory and can be found in lastpass

## Building

### With Docker:

`docker build -t coco/v1-orgs-transformer .`

`docker run -ti --env BASE_URL=<base url> --env BERTHA_SERVICE_URL=<bertha url> --env TME_BASE_URL=<structure service url> --env TME_USERNAME=<user> --env TME_PASSWORD=<pass> --env TOKEN=<token> --env CACHE_FILE_NAME=<file> coco/v1-orgs-transformer`

## Endpoints

### GET /transformers/authors
The V1 Authors transformer holds all the V1 Authors in memory and this endpoint gets the JSON for ALL the Authors. Useful for piping to a file  or using with up-rest-utils but be careful using this via Postman or a Browser as it is a lot of JSON

A successful GET results in a 200. 

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-authors-transformer/transformers/authors`

### GET /transformers/authors/{uuid}
The V1 Authors transformer holds all the V1 Authors in memory and this endpoint gets the JSON for an author with a given UUID. The UUID is derived from the TME composite id at this point

A successful GET results in a 200 and 404 for not finding the author

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-authors-transformer/transformers/authors/8138ca3f-b80d-3ef8-ad59-6a9b6ea5f15e`

### GET /transformers/authors/__ids

All of the UUIDS for ALL the V1 authors - This is needed for loading via the concept publisher

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-authors-transformer/transformers/authors/__ids`

### GET /transformers/authors/__count
A count of how authors are in the transformer's memory cache

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-authors-transformer/transformers/authors/__count`


### POST /transformers/authors/__reload 

Fetches all the V1 Authors from TME and reloads the cache. There is no payload for this post

`curl -X POST https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-authors-transformer/transformers/authors/__reload`

### Admin endpoints
Healthchecks: [http://localhost:8080/__health](http://localhost:8080/__health)

Ping: [http://localhost:8080/ping](http://localhost:8080/ping) or [http://localhost:8080/__ping](http://localhost:8080/__ping)

Build-info: [http://localhost:8080/build-info](http://localhost:8080/build-info) 

Good to Go: [http://localhost:8080/__gtg](http://localhost:8080/__gtg) 

### API Document  
[V1 Authors Transformer API Endpoints](https://docs.google.com/document/d/1-Eyhs98a3J1zw5OHfFZ0uXzyFCywBKnvC3RmrBc29cU)
