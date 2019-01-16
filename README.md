# Invoicer

.. is a CLI tool written in Go which prints invoices in PDF. Configured by templates and arguments.

This is like the smallest amount of logic I needed to generate invoices.

## Build 

Install Golang, adn then in the root of this repo:

```
$ go build
```


## Usage

```
./invoicer --currency EUR  --vatproc 22 --templ templates/example.yml
           --amount 2888   --period "November, December 2018" 
           --due "2019-01-30" --date "2019-01-14" 
           --descpri "November work | 1000 EUR"
           --descpri "December work | 1888 EUR" 
           --descpri "Total | 2888 EUR"
           --worklog "https://gist.github.com/t0mk/67046a598f7f1615cce3b31f5ad9b313" 
           --outfile "example.pdf"
```

It will generate pdf to [example.pdf](example.pdf).


## Reference number 

Reference numbers are generated according to ISO 11649 creditor reference.

However, you can pass `--ref` arg which will populate the ref by given string.

## Barcode

It generates the barcode for Finnish bank apps and whatnot. I did this some time ago, there is some sort of standard to it, I don't know where.


