# ftb

fulltext search app back-end

## configuration

Edit `config.toml`.

options | type | descr
---|---|---
ResetES | bool | if true, recreate ES indices
ESAddresses | []string | ES addresses
IndexName | string | ES index name
MecabDir | string | base path for mecab unidic dictionaries
BulkSourceDir | string | base path for files to be indexed
BulkESUnitNum | string | max unit size for bulk indexing
IsBulkSubdir | bool | if true, e.g., '0001-001001' is treated as '0001/0001-001001'
AbortOnError | bool | if true, abort on error


## dev

```sh
  source envfile
  air
```

## test

```sh
  source envfile
  # go generate
  go test
```


## build

```sh
  source envfile
  # go generate
  go build
```

