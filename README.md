# V1 Authors Transformer
[![CircleCI](https://circleci.com/gh/Financial-Times/v1-authors-transformer.svg?style=svg)](https://circleci.com/gh/Financial-Times/v1-authors-transformer) [![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/v1-authors-transformer)](https://goreportcard.com/report/github.com/Financial-Times/v1-authors-transformer) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/v1-authors-transformer/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/v1-authors-transformer?branch=master) [![codecov](https://codecov.io/gh/Financial-Times/v1-authors-transformer/branch/master/graph/badge.svg)](https://codecov.io/gh/Financial-Times/v1-authors-transformer)

The service exposes read API endpoints to get information about authors contributing to FT content.
It pulls and transforms V1/TME Authors into the UPP JSON model of an Author.  
Raw author data is pulled from 2 sources: 
* from TME "authors" taxonomy 
* from a Google spreadsheet via [Bertha API](https://github.com/ft-interactive/bertha/wiki/Tutorial).  
 
The authors pulled from Bertha API are effectively the same V1/TME authors but they are embellished with extra information which editors update manually in the spreadsheet and therefore they are also known as curated authors.

## Installation

For the first time:

`go get github.com/Financial-Times/v1-authors-transformer`

or update:

`go get -u github.com/Financial-Times/v1-authors-transformer`

## Running locally
````
$GOPATH/bin/ ./v1-authors-transformer --tme-username=<TME_USERNAME> --tme-password=<TME_PASSWORD> --token=<TOKEN> --bertha-source-url=<BERTHA_SOURCE_URL>
````
or 
````
$GOPATH/bin/v1-authors-transformer TME_USERNAME=<TME_USERNAME> TME_PASSWORD=<TME_PASSWORD>  TOKEN=<TOKEN> BERTHA_SOURCE_URL=<BERTHA_SOURCE_URL> v1-authors-transformer
````
The latter run command does't perform parameter checking.

TME credentials are mandatory and can be found in lastpass.  
BERTHA_SOURCE_URL can be found in [UPP Documentation about curated authors](https://sites.google.com/a/ft.com/universal-publishing/how-can-i/load-curated-authors?pli=1) in Google sites.  
Refer to the full list of initialisation parameters whose defaults you can override in the code of main.go  
TME_BASE_URL by default is pointing to prod TME, you may consider using TME instance in test  [https://test-tme.ft.com]  

## Building

### With Docker:

````
docker build -t coco/v1-authors-transformer .  
docker run -ti --env BASE_URL=<base url> --env BERTHA_SOURCE_URL=<bertha url> --env TME_BASE_URL=<structure service url> --env TME_USERNAME=<user> --env TME_PASSWORD=<pass> --env TOKEN=<token> --env CACHE_FILE_NAME=<file> coco/v1-authors-transformer  
````
## Endpoints

### GET /transformers/authors
The V1 Authors transformer holds all the V1 Authors in memory and this endpoint gets the JSON for ALL the Authors. Useful for piping to a file  or using with up-rest-utils but be careful using this via Postman or a Browser as it is a lot of JSON

A successful GET results in a 200. 

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-authors-transformer/transformers/authors`

### GET /transformers/authors/{uuid}
The V1 Authors transformer holds all the V1 Authors in memory and this endpoint gets the JSON for an author with a given UUID. The UUID is derived from the TME composite id at this point

A successful GET results in a 200 and 404 for not finding the author

`curl -X GET https://{pub-semantic-user}:{pub-semantic-password}@semantic-up.ft.com/__v1-authors-transformer/transformers/authors/fe11b796-2538-3bf5-85b6-cd88c707a972`

Here is a response example for a non-curated author.
````
{
  "uuid": "fe11b796-2538-3bf5-85b6-cd88c707a972",
  "prefLabel": "Lara Feigel",
  "type": "Person",
  "alternativeIdentifiers": {
    "TME": [
      "ZmY4MzJmZDYtZjE5My00ZTM3LWE4ZDEtNTgxZTE0YWZkYWNl-QXV0aG9ycw=="
    ],
    "uuids": [
      "fe11b796-2538-3bf5-85b6-cd88c707a972"
    ]
  },
  "aliases": [
    "Lara Feigel"
  ]
}
````

Here is a response example for a curated author.

```
{
  "uuid": "0f07d468-fc37-3c44-bf19-a81f2aae9f36",
  "prefLabel": "Martin Wolf",
  "type": "Person",
  "alternativeIdentifiers": {
    "TME": [
      "Q0ItMDAwMDkwMA==-QXV0aG9ycw=="
    ],
    "uuids": [
      "0f07d468-fc37-3c44-bf19-a81f2aae9f36"
    ]
  },
  "aliases": [
    "Martin Wolf"
  ],
  "name": "Martin Wolf",
  "emailAddress": "martin.wolf@ft.com",
  "twitterHandle": "@martinwolf_",
  "description": "Martin Wolf is chief economics commentator at the Financial Times, London. He was awarded the CBE (Commander of the British Empire) in 2000 “for services to financial journalism”.",
  "descriptionXML": "<p>Martin Wolf is chief economics commentator at the Financial Times, London. He was awarded the CBE (Commander of the British Empire) in 2000 “for services to financial journalism”.</p>",
  "_imageUrl": "https://www.ft.com/__origami/service/image/v2/images/raw/fthead:martin-wolf?source=next"
}
```
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

Build-info: [http://localhost:8080/__build-info](http://localhost:8080/__build-info) 

Good to Go: [http://localhost:8080/__gtg](http://localhost:8080/__gtg) 

### API Document  
[V1 Authors Transformer API Endpoints](https://docs.google.com/document/d/1-Eyhs98a3J1zw5OHfFZ0uXzyFCywBKnvC3RmrBc29cU)
