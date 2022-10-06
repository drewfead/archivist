Archivist
=========

This project consists of useful-to-me CLIs for interacting with the wide world of movie and physical media data.

Building
--------

Use the make target!

```
make build
```

Discs
-----

Discs currently supports interactive blu-ray searches by scraping blu-ray.com

```
bin/discs search -i <searchTerm>
```

Support for vim bindings in interactive mode using 
```
bin/discs search -i --vim <searchTerm>
```

See more using:

```
bin/discs --help
```