# V1 People Transformer
[![CircleCI](https://circleci.com/gh/Financial-Times/v1-people-transformer.svg?style=svg)](https://circleci.com/gh/Financial-Times/v1-people-transformer) [![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/v1-people-transformer)](https://goreportcard.com/report/github.com/Financial-Times/v1-people-transformer) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/v1-people-transformer/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/v1-people-transformer?branch=master)

An API for pulling in and transforming V1/TME People into the UPP representation of a Person 

## Installation

For the first time:

`go get github.com/Financial-Times/v1-people-transformer`

or update:

`go get -u github.com/Financial-Times/v1-people-transformer`

## Running

`$GOPATH/bin/v1-people-transformer --tme-username={tme-username} --port={port} --tme-password={tme-password} --token={tme-token} --base-url={base-url for the app} --tme-base-url={tme-base-url} --maxRecords={maxRecords} --batchSize={batchSize} --cache-file-name={cache-file-name}`

TME credentials are mandatory and can be found in lastpass

## Building

### With Docker:

`docker build -t coco/v1-orgs-transformer .`

`docker run -ti --env BASE_URL=<base url> --env TME_BASE_URL=<structure service url> --env TME_USERNAME=<user> --env TME_PASSWORD=<pass> --env TOKEN=<token> --env CACHE_FILE_NAME=<file> coco/v1-orgs-transformer`

## Endpoints

### GET /transformers/people
The V1 people transformer holds all the V1 People in memory and this endpoint gets the JSON for ALL the people. Useful for piping to a file  or using with up-rest-utils but be careful using this via Postman or a Browser as it is a lot of JSON

A successful GET results in a 200. 

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-people-transformer/transformers/people`

### GET /transformers/people/{uuid}
The V1 people transformer holds all the V1 People in memory and this endpoint gets the JSON a person with a given UUID. The UUID is derived from the TME composite id at this point

A successful GET results in a 200 and 404 for not finding the person

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-people-transformer/transformers/people/8138ca3f-b80d-3ef8-ad59-6a9b6ea5f15e`

### GET /transformers/people/__ids

All of the UUIDS for ALL the V1 people - This is needed for loading via the concept publisher

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-people-transformer/transformers/people/__ids`

### GET /transformers/people/__count
A count of how people are in the transformer's memory cache

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-people-transformer/transformers/people/__count`


### POST /transformers/people/__reload 

Fetches all the V1 people from TME and reloads the cache. There is no payload for this post

`curl -X POST https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-people-transformer/transformers/people/__reload`

### Admin endpoints
Healthchecks: [http://localhost:8080/__health](http://localhost:8080/__health)

Ping: [http://localhost:8080/ping](http://localhost:8080/ping) or [http://localhost:8080/__ping](http://localhost:8080/__ping)

Build-info: [http://localhost:8080/build-info](http://localhost:8080/build-info) 

Good to Go: [http://localhost:8080/__gtg](http://localhost:8080/__gtg) 

