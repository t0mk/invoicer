package main

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

type PaymentInfo struct {
	Account         string
	ReferenceNumber string
	Due             string
	Amount          string
	Vat             string
	VatProc         int64
	Total           string
	Barcode         string
	Currency        string
}

type Invoice struct {
	Name             string `yaml:"name"`
	Tldr             string
	From             string
	For              string
	InvoiceID        string
	InvoiceDate      string
	Payment          PaymentInfo
	PricingAgreement string `yaml:"pricingAgreement"`
	BilledWork       string `yaml:"billedWork"`
}

func readfile(p string) []byte {
	filename, _ := filepath.Abs("./" + p)
	content, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}
	return content
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func genref(base string) string {
	if len(base) == 0 {
		panic("you must give some base")
	}

	multiplier := [3]int{7, 3, 1}
	t := 0
	esab := reverse(base)
	for i, length := 0, len(esab); i < length; i++ {
		n, err := strconv.Atoi(string(esab[i]))
		if err != nil {
			panic(fmt.Sprintf("cant read number %s ", string(esab[i])))
		}
		t += n * multiplier[i%3]
	}
	check_digit := (((t % 10) * 10) - t) % 10
	if check_digit < 0 {
		check_digit += 10
	}

	referenceNumber := base + strconv.Itoa(check_digit)
	return referenceNumber
}

func stripchars(str, chr string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(chr, r) < 0 {
			return r
		}
		return -1
	}, str)
}

func getDestID(c string) uint32 {
	// This is to distinguish customers in the reference number,
	// it gets 3 digit "hash" based on the client (template name)
	crc32q := crc32.MakeTable(0xD5828281)
	clientHash := crc32.Checksum([]byte(c), crc32q)
	clientID := clientHash % 1000
	return clientID
}

func barcode(iban, eur, refnum, ddate string) string {
	// This is something in Finland... I don't even remember where I
	// learned this from
	codeIBAN := "4" + iban[2:]
	codeEUR := fmt.Sprintf("%08s", stripchars(eur, "."))
	codeRefNum := fmt.Sprintf("%023s", refnum)
	codeDate := stripchars(ddate, "-")[2:]
	return codeIBAN + codeEUR + codeRefNum + codeDate

}

func invPrint(c *cli.Context) error {
	templ := c.String("templ")
	cur := c.String("currency")
	date := c.String("date")
	due := c.String("due")
	amount := c.Float64("amount")
	vatproc := c.Int64("vatproc")
	period := c.String("period")
	descpri := c.StringSlice("descpri")
	bw := `
| **description**  | **price** |
|------------------|-----------|
`
	for _, dp := range descpri {
		bw += fmt.Sprintf("| %s |\n", dp)
	}

	yamlFile := readfile(templ)

	var i Invoice

	err := yaml.Unmarshal(yamlFile, &i)
	client := i.Name
	if err != nil {
		panic(err)
	}

	vat := amount * (float64(vatproc) / 100)
	total := amount + vat

	i.InvoiceDate = date
	i.Payment.Currency = cur
	clientID := getDestID(client)
	idate := stripchars(date, "-")
	base := idate + strconv.FormatUint(uint64(clientID), 10) + strconv.Itoa(random(10, 99))
	i.Payment.ReferenceNumber = genref(base)
	i.InvoiceID = i.Payment.ReferenceNumber
	i.Payment.Due = due
	i.BilledWork = bw
	totalPrintable := strconv.FormatFloat(total, 'f', 2, 64)

	i.Payment.Total = totalPrintable
	i.Payment.Amount = strconv.FormatFloat(amount, 'f', 2, 64)
	i.Payment.Vat = strconv.FormatFloat(vat, 'f', 2, 64)
	i.Payment.VatProc = vatproc
	i.Payment.Barcode = barcode(i.Payment.Account, totalPrintable,
		i.Payment.ReferenceNumber, i.Payment.Due)

	i.Tldr = fmt.Sprintf("For my contract work in %s, I invoice you for %s %s payable to %s, ref. nr. %s by %s", period, i.Payment.Total, i.Payment.Currency, i.Payment.Account, i.Payment.ReferenceNumber, i.Payment.Due)

	printMarkDown(i)
	return nil
}

func printMarkDown(i Invoice) {
	fmt.Println("# Invoice", i.InvoiceID)
	fmt.Println()
	fmt.Println("## From")
	fmt.Println()
	fmt.Println("```")
	fmt.Println(i.From)
	fmt.Println("```")
	fmt.Println("## For")
	fmt.Println()
	fmt.Println("```")
	fmt.Println(i.For)
	fmt.Println("```")
	fmt.Println("## Summary")
	fmt.Println()
	fmt.Println("|   |   |")
	fmt.Println("|---|---|")
	fmt.Println("| Invoice date |", i.InvoiceDate, "|")
	fmt.Println("| Invoice ID |", i.InvoiceID, "|")
	fmt.Println("")
	fmt.Println(i.Tldr)
	fmt.Println("")
	fmt.Println("## Pricing agreement")
	fmt.Println("")
	fmt.Println(i.PricingAgreement)
	fmt.Println("")
	fmt.Println("## Billed work")
	fmt.Println("")
	fmt.Println(i.BilledWork)
	fmt.Println("")
	fmt.Println("\\pagebreak")
	fmt.Println("")
	fmt.Println("### Detailed amounts")
	fmt.Println("")
	fmt.Println("|   |   |")
	fmt.Println("|---|---|")
	fmt.Println("| Amount without VAT |", i.Payment.Amount, i.Payment.Currency, "|")
	fmt.Println("| VAT", i.Payment.VatProc, "% |", i.Payment.Vat, i.Payment.Currency, "|")
	fmt.Println("| Total Amount |", i.Payment.Total, i.Payment.Currency, " |")
	fmt.Println("")
	fmt.Println("## Payment information")
	fmt.Println("")
	fmt.Println("|   |   |")
	fmt.Println("|---|---|")
	fmt.Println("| Amount to pay |", i.Payment.Total, i.Payment.Currency, " |")
	fmt.Println("| Ref. number |", i.Payment.ReferenceNumber, "|")
	fmt.Println("| Due date |", i.Payment.Due, "|")
	fmt.Println("| IBAN |", i.Payment.Account, "|")
	fmt.Println("")
	if strings.HasPrefix(i.Payment.Account, "FI") {
		fmt.Println("In Finland, you can also copypaste following \"barcode\" to your Internet banking payment form:")
		fmt.Println("")
		fmt.Println(i.Payment.Barcode)
	}

}

func main() {
	now := time.Now()
	app := &cli.App{
		Action: invPrint,
		Name:   "inv",
		Usage:  "generates invoice in MarkDown",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "templ"},
			&cli.Float64Flag{Name: "amount"},
			&cli.Int64Flag{Name: "vatproc", Value: 0},
			&cli.StringFlag{Name: "period"},
			&cli.StringSliceFlag{Name: "descpri"},
			&cli.StringFlag{Name: "currency", Value: "EUR"},
			&cli.StringFlag{Name: "due", Value: now.Add(336 * time.Hour).String()[:10]},
			&cli.StringFlag{Name: "date", Value: now.String()[:10]},
		},
	}
	app.Run(os.Args)
}
