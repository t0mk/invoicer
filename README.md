# Invoicer

.. is a CLI tool written in Go which prints invoices in MarkDown. Configured by templates and arguments.

This is like the smallest amount of logic I needed to generate invoices.

It's possible to generate pdf from markdown, see below.

I was thinking to generate pdf straight from go, it's poss (https://github.com/jung-kurt/gofpdf), but
then I thought that if I keep it in MarkDown, I can browse the invoices in HTTP interfaces to git repos -
github, bitbucket, gogs all render MarkDown real nicely.


## Usage

```
./invoicer --vatproc 22  --templ templates/example.yml --amount 8800 \
          --period "January 2018"  --date "2018-01-30" --due "2018-02-07" \
          --descpri "100 hours: consultations and setup | 8800 EUR"
```

It will print MarkDown to stdout.

## Generate PDF

I generate pdfs with pandoc. Install pandoc (might take a while), and then
just pass generated md file to pandoc.

```
./invoicer --vatproc 22  --templ templates/example.yml --amount 8800 \
          --period "January 2018"  --date "2018-01-30" --due "2018-02-07" \
          --descpri "100 hours: consultation and setup | 8800 EUR" > ex_0118.md
pandoc ex_0118.md -o ex_0118.pdf --columns 150 -V geometry:margin=1in
```

## Rererence number 

Refence numbers are generated accoiding to ISO 11649 creditor reference.

However, you can pass `--ref` arg which will populate the ref by given string.

## Examples

In this repo:

* [ex_0118.md](ex_0118.md)
* [ex_0118.pdf](ex_0118.pdf)
