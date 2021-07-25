module github.com/umputun/feed-master

go 1.16

require (
	github.com/ChimeraCoder/anaconda v2.0.0+incompatible
	github.com/ChimeraCoder/tokenbucket v0.0.0-20131201223612-c5a927568de7 // indirect
	github.com/azr/backoff v0.0.0-20160115115103-53511d3c7330 // indirect
	github.com/denisbrodbeck/striphtmltags v6.6.6+incompatible
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/didip/tollbooth_chi v0.0.0-20170928041846-6ab5f3083f3d
	github.com/dustin/go-jsonpointer v0.0.0-20160814072949-ba0abeacc3dc // indirect
	github.com/dustin/gojson v0.0.0-20160307161227-2e71ec9dd5ad // indirect
	github.com/garyburd/go-oauth v0.0.0-20180319155456-bca2e7f09a17 // indirect
	github.com/go-chi/chi/v5 v5.0.3
	github.com/go-chi/render v1.0.1
	github.com/go-pkgz/lcw v0.8.1
	github.com/go-pkgz/lgr v0.10.4
	github.com/go-pkgz/rest v1.9.2
	github.com/go-pkgz/syncs v1.1.1
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jessevdk/go-flags v1.4.0
	github.com/microcosm-cc/bluemonday v1.0.9
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/tcolgate/mp3 v0.0.0-20170426193717-e79c5a46d300
	github.com/xelaj/mtproto v1.0.0
	go.etcd.io/bbolt v1.3.5
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	gopkg.in/yaml.v2 v2.2.8
)

// mtproto v1.0.1 is not released and master is not working properly,
// so we're using the fork with goroutines leak fixed
replace github.com/xelaj/mtproto => github.com/paskal/mtproto v1.0.1-bugfix
