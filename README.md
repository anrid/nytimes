# Indexing NY Times Articles in Elasticsearch

## Usage

Start a 3-node ES 7.x cluster:

```bash
$ docker compose -f deployments/es7-3n-compose.yml up -d
```

Fetch NY Times articles from their public API, starting from Dec 2022:

```bash
$ go run cmd/fetch/main.go --year 2022 --month 12

Fetching URL: https://api.nytimes.com/svc/archive/v1/2022/12.json?api-key=
Wrote 19477556 / 19477556 compressed bytes to file data/articles-2022-12.json.gz
Sleeping for 6 sec to avoid rate limit ..
Fetching URL: https://api.nytimes.com/svc/archive/v1/2023/1.json?api-key=
Wrote 18589680 / 18589680 compressed bytes to file data/articles-2023-1.json.gz
Sleeping for 6 sec to avoid rate limit ..
Fetching URL: https://api.nytimes.com/svc/archive/v1/2023/2.json?api-key=
Wrote 15873102 / 15873102 compressed bytes to file data/articles-2023-2.json.gz
We're at 2023-2 - we're all caught up in time! Donezo!

# You now have a bunch of *.json.gz files full of articles in ./data
```

Index NY Times articles in ES:

```bash
$ go run cmd/load/main.go --create-index --start-from 2022-12

> HEAD http://localhost:9200/?error_trace=true
Pinged ES successfully
> DELETE http://localhost:9200/nytimes-articles?ignore_unavailable=true&pretty=true
Deleted existing index `nytimes-articles` (status: 200)
Created new index `nytimes-articles` (status: 200)
Reading dir: data/
Indexed 9880 articles total
Done. Read 3 files in 2.013946478s
```

Run benchmark against the new ES index:

```bash
# Query command usage:
$ go run cmd/query/main.go --help

Usage of /cmd/query/main:
      --cache          enable search engine caching (default true)
      --count int      number of calls to search engine (default 10)
      --dump           dump search engine result of first query
      --index string   search engine index name (default "nytimes-articles")
      --query string   query to run (path to JSON file) (default "./assets/mappings/nytimes/query-simple.json")
      --threads int    number of threads to run benchmark in concurrently (default 10)

# Run a simple benchmark, 10 iterations across 10 threads:
$ go run cmd/query/main.go

> HEAD http://localhost:9200/?error_trace=true
Pinged ES successfully
Query payload:
{
  "query": {
    "bool": {
      "must": [
        {
          "match": {
            "headline": "President"
          }
        }
      ]
    }
  }
}

> GET http://localhost:9201/_stats
Completed first request in 98.946383ms
[Thread 10] running benchmark, 10 iterations
[Thread 02] running benchmark, 10 iterations
[Thread 01] running benchmark, 10 iterations
[Thread 03] running benchmark, 10 iterations
[Thread 08] running benchmark, 10 iterations
[Thread 06] running benchmark, 10 iterations
[Thread 05] running benchmark, 10 iterations
[Thread 07] running benchmark, 10 iterations
[Thread 09] running benchmark, 10 iterations
[Thread 04] running benchmark, 10 iterations
[Thread 10] completed 10 requests in 188.858577ms (52.9 req/s) | min: 4.937454ms / max: 80.216712ms / mean: 8.889573ms
[Thread 06] completed 10 requests in 194.420731ms (51.4 req/s) | min: 5.151222ms / max: 72.469707ms / mean: 7.086113ms
[Thread 05] completed 10 requests in 213.133434ms (46.9 req/s) | min: 3.867028ms / max: 138.370538ms / mean: 7.039386ms
[Thread 08] completed 10 requests in 215.596919ms (46.4 req/s) | min: 4.45052ms / max: 77.41287ms / mean: 8.158992ms
[Thread 01] completed 10 requests in 216.713314ms (46.1 req/s) | min: 5.253569ms / max: 110.809016ms / mean: 8.763341ms
[Thread 04] completed 10 requests in 216.318873ms (46.2 req/s) | min: 3.782726ms / max: 141.199454ms / mean: 7.196361ms
[Thread 09] completed 10 requests in 219.484012ms (45.6 req/s) | min: 4.46494ms / max: 108.46021ms / mean: 11.587092ms
[Thread 02] completed 10 requests in 230.458599ms (43.4 req/s) | min: 3.377986ms / max: 155.998477ms / mean: 6.732632ms
[Thread 07] completed 10 requests in 236.990408ms (42.2 req/s) | min: 4.508526ms / max: 150.563158ms / mean: 8.163512ms
[Thread 03] completed 10 requests in 240.853723ms (41.5 req/s) | min: 3.70519ms / max: 156.083115ms / mean: 7.352016ms
Done. Completed 100 requests total in 241.141037ms (414.7 req/s)
Query cache   : +0   hits / +0   miss
Request cache : +98  hits / +3   miss
```

## Data

The dataset is all NY Times articles since the Jan 1852, fetched from https://developer.nytimes.com/apis. A typical article looks as follows:

```json
{
  "_id": "nyt://article/ab920e95-4eb9-561c-84c8-ecb5286d3766",
  "abstract": "On the pleasures and pains of joining up with other people after a long, quiet time in the Covid doldrums.",
  "byline": {
    "organization": "",
    "original": "By Andy Miller",
    "person": [
      {
        "firstname": "Andy",
        "lastname": "Miller"
      }
    ]
  },
  "headline": {
    "main": "Hello, World. It’s Been a While.",
    "print_headline": "Trying to Chug Along but Going Off the Rails"
  },
  "keywords": [
    {
      "name": "subject",
      "rank": 1,
      "value": "Books and Literature"
    },
    {
      "name": "organizations",
      "rank": 2,
      "value": "Arsenal (Soccer Team)"
    },
    {
      "name": "persons",
      "rank": 3,
      "value": "Wilde, Oscar"
    },
    {
      "name": "persons",
      "rank": 4,
      "value": "Everett, Rupert"
    },
    {
      "name": "glocations",
      "rank": 5,
      "value": "Great Britain"
    },
    {
      "name": "subject",
      "rank": 6,
      "value": "Coronavirus (2019-nCoV)"
    }
  ],
  "lead_paragraph": "I am traveling on a train, reading a book, glad to be alive.",
  "multimedia": [
    {
      "url": "images/2022/08/07/fashion/31EPISODE-MILLER/31EPISODE-MILLER-articleLarge.jpg",
      "width": 600,
      "height": 600,
      "subType": "xlarge"
    },
    {
      "url": "images/2022/08/07/fashion/31EPISODE-MILLER/31EPISODE-MILLER-thumbStandard.jpg",
      "width": 75,
      "height": 75,
      "subType": "thumbnail"
    },
    {
      "url": "images/2022/08/07/fashion/31EPISODE-MILLER/31EPISODE-MILLER-thumbLarge.jpg",
      "width": 150,
      "height": 150,
      "subType": "thumbLarge"
    }
  ],
  "pub_date": "2022-08-01T00:00:09+0000"
}
```

# Generate data based on NY Times articles

```bash
# Generate 10 sentences with max 12 words each based on 10,000 NY Times articles.
# The data follows the natural word distribution of the articles and uses a
# "word graph" to ensure that words follow each other naturally, i.e. that a
# word can naturally occur after another (as opposed to a completely random word).
#
$ go run cmd/datagen/main.go --dir ./data --start-from articles-2022 --max-docs 10_000 --num 10 --max-words 12

navajo nation had announced on the nonprofit organization pen lope cruz by a
criminally responsible by brands are so much of closure in case that lets
nocturna’ review do visitors had provided the shop is there are cops who
kurson kushner . but when amina begum stares at more daunting final challenge
wanes after the jan . when the annual state of forbes avenue raised
badder androids dream after the texas spike in this newsletter i speak and
parton eminem planned meeting with black skinhead at the facebook illumina now there
wants. . the winter . the nose the 1 million followers who attempted
symonds . bp as ambitious ambassadors of the breach in the day buffeted
cream-free creamy minty allure offering developed more leverage his front lines . said
```