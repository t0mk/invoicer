package main

import (
	"fmt"
	"hash/crc32"
	"image/gif"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/jung-kurt/gofpdf"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var strToNum = map[string]string{
	"A": "10",
	"B": "11",
	"C": "12",
	"D": "13",
	"E": "14",
	"F": "15",
	"G": "16",
	"H": "17",
	"I": "18",
	"J": "19",
	"K": "20",
	"L": "21",
	"M": "22",
	"N": "23",
	"O": "24",
	"P": "25",
	"Q": "26",
	"R": "27",
	"S": "28",
	"T": "29",
	"U": "30",
	"V": "31",
	"W": "32",
	"X": "33",
	"Y": "34",
	"Z": "35",
}

type PaymentInfo struct {
	Account         string
	Swift           string
	Bankaddress     string
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
	DescPri          []string
	Worklog          string
	PO               string
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

func replace(s string) string {
	r := s
	for k, v := range strToNum {
		r = strings.Replace(r, k, v, -1)
	}
	return r
}

func getUInt(s string) uint64 {
	ui, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return ui
}

func getChecksum(s string) string {
	pre := s + "RF00"
	pre = replace(pre)
	cs := 98 - (getUInt(pre) % 97)
	return fmt.Sprintf("%02d", cs)

}

func genRef(r string) string {
	// international reference due to ISO 11649 creditor reference
	ur := strings.ToUpper(r)
	nr := replace(ur)
	cs := getChecksum(nr)
	ref := "RF" + cs + ur
	validateRef(ref)
	return ref
}

func validateRef(r string) {
	rr := strings.ToUpper(r)
	wr := rr[4:len(rr)] + rr[0:4]
	nr := replace(wr)
	if len(rr) >= 26 {
		panic("bad length")
	}
	mod := getUInt(nr) % 97
	if mod != 1 {
		panic("bad mod")
	}

}

func genref(base string) string {
	// finnish domestic refnum
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

func barcod(iban, eur, refnum, ddate string) string {
	// This is something in Finland... I don't even remember where I
	// learned this from
	codeIBAN := "4" + iban[2:]
	codeEUR := fmt.Sprintf("%08s", stripchars(eur, "."))
	codeRefNum := fmt.Sprintf("%023s", refnum)
	codeDate := stripchars(ddate, "-")[2:]
	return codeIBAN + codeEUR + codeRefNum + codeDate

}

func codeToFile(c, fn string) error {
	e, err := code128.Encode(c)
	if err != nil {
		return err
	}

	cd, err := barcode.Scale(e, 800, 200)
	if err != nil {
		return err
	}

	file, _ := os.Create(fn)
	defer file.Close()

	gif.Encode(file, cd, &gif.Options{NumColors: 256})
	return nil

}

func invPrint(c *cli.Context) error {
	templ := c.String("templ")
	yamlFile := readfile(templ)

	var i Invoice

	err := yaml.Unmarshal(yamlFile, &i)
	client := i.Name
	if err != nil {
		panic(err)
	}
	i.Worklog = c.String("worklog")
	cur := c.String("currency")
	date := c.String("date")
	due := c.String("due")
	amount := c.Float64("amount")
	vatproc := c.Int64("vatproc")
	period := c.String("period")
	ref := c.String("ref")
	descpri := c.StringSlice("descpri")
	outfile := c.String("outfile")
	if !strings.HasSuffix(outfile, ".pdf") {
		panic(fmt.Errorf("outfile should end with .pdf"))
	}

	i.DescPri = descpri

	vat := amount * (float64(vatproc) / 100)
	total := amount + vat

	i.InvoiceDate = date
	i.PO = "3280075065"
	i.Payment.Currency = cur
	clientID := getDestID(client)
	if len(ref) == 0 {
		base := strconv.FormatUint(uint64(clientID), 10) + strconv.Itoa(random(10, 99))
		ref = genRef(genref(base))
	}

	i.Payment.ReferenceNumber = ref
	i.InvoiceID = i.Payment.ReferenceNumber
	i.Payment.Due = due
	totalPrintable := strconv.FormatFloat(total, 'f', 2, 64)

	i.Payment.Total = totalPrintable
	i.Payment.Amount = strconv.FormatFloat(amount, 'f', 2, 64)
	i.Payment.Vat = strconv.FormatFloat(vat, 'f', 2, 64)
	i.Payment.VatProc = vatproc
	i.Payment.Barcode = barcod(i.Payment.Account, totalPrintable,
		i.Payment.ReferenceNumber, i.Payment.Due)

	i.Tldr = fmt.Sprintf("I am a contractor working on Equinix Metal Terraform provider and Golang SDK. For my contract work in %s, I invoice you for %s %s payable to %s, ref. nr. %s by %s.", period, i.Payment.Total, i.Payment.Currency, i.Payment.Account, i.Payment.ReferenceNumber, i.Payment.Due)

	pdf, err := getPdf(&i)
	if err != nil {
		panic(err)
	}
	err = pdf.OutputFileAndClose(outfile)
	if err != nil {
		panic(err)
	}

	return nil
}

func main() {
	now := time.Now()
	app := &cli.App{
		Action: invPrint,
		Name:   "inv",
		Usage:  "generates PDF invoice",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "templ"},
			&cli.Float64Flag{Name: "amount"},
			&cli.Int64Flag{Name: "vatproc", Value: 0},
			&cli.StringFlag{Name: "period"},
			&cli.StringFlag{Name: "worklog"},
			&cli.StringFlag{Name: "outfile"},
			&cli.StringSliceFlag{Name: "descpri"},
			&cli.StringFlag{Name: "currency", Value: "EUR"},
			&cli.StringFlag{Name: "ref"},

			&cli.StringFlag{Name: "due", Value: now.Add(336 * time.Hour).String()[:10]},
			&cli.StringFlag{Name: "date", Value: now.String()[:10]},
		},
	}
	app.Run(os.Args)
}

func getPdf(c *Invoice) (*gofpdf.Fpdf, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetXY(20, 10)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(10, 30, "Invoice "+c.InvoiceID)

	pdf.SetXY(20, 20)
	pdf.SetFont("Helvetica", "BI", 13)
	pdf.Cell(10, 30, "Invoice From")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(20, 40)
	pdf.MultiCell(100, 5, c.From, "", "", false)

	pdf.SetXY(120, 20)
	pdf.SetFont("Helvetica", "BI", 13)
	pdf.Cell(10, 30, "Invoice For")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(120, 40)
	pdf.MultiCell(100, 5, c.For, "", "", false)

	pdf.SetXY(20, 65)
	pdf.SetFont("Helvetica", "BI", 13)
	pdf.Cell(10, 30, "Summary")

	pdf.SetFont("Helvetica", "I", 10)
	pdf.SetXY(20, 85)
	pdf.Cell(100, 5, "Invoice Date:")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(55, 85)
	pdf.Cell(100, 5, c.InvoiceDate)

	pdf.SetFont("Helvetica", "I", 10)
	pdf.SetXY(20, 90)
	pdf.Cell(100, 5, "Invoice ID:")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(55, 90)
	pdf.Cell(100, 5, c.InvoiceID)

	pdf.SetFont("Helvetica", "BI", 10)
	pdf.SetXY(20, 95)
	pdf.Cell(100, 5, "Purchase Order:")

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(55, 95)
	pdf.Cell(100, 5, c.PO)

	pdf.SetFont("Helvetica", "I", 10)
	pdf.SetXY(20, 100)
	pdf.Cell(100, 5, "Pricing Agreement:")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(55, 100)
	pdf.Cell(100, 5, c.PricingAgreement)

	pdf.SetXY(20, 107)
	pdf.MultiCell(150, 5, c.Tldr, "", "", false)

	pdf.SetXY(20, 112)
	pdf.SetFont("Helvetica", "BI", 13)
	pdf.Cell(10, 30, "Billed Work")

	pdf.SetFont("Helvetica", "B", 10)

	pdf.SetXY(20, 120.5)
	pdf.Cell(10, 30, "Description")

	pdf.SetXY(105, 120.5)
	pdf.Cell(10, 30, "Price")

	pdf.SetFont("Helvetica", "", 10)
	y := 145.0

	pdf.Line(21, 132, 122, 132)
	pdf.Line(21, 139, 122, 139)
	for _, i := range c.DescPri {
		j := strings.Split(i, " | ")
		pdf.SetXY(20, y)
		pdf.Cell(100, 5, j[0])
		pdf.SetXY(105, y)
		pdf.Cell(100, 5, j[1])
		y += 5
	}
	pdf.Line(21, y+1, 122, y+1)

	pdf.SetXY(20, y+3)

	pdf.Cell(27, 5, "Log of my work:")
	pdf.WriteLinkString(5, c.Worklog, c.Worklog)

	pdf.SetXY(20, y+5)
	pdf.SetFont("Helvetica", "BI", 13)
	pdf.Cell(10, 30, "Payment information")

	pdf.SetFont("Helvetica", "I", 10)

	pdf.Line(21, y+24, 97, y+24)
	pdf.Line(121, y+24, 182, y+24)

	pdf.Line(21, y+56, 120, y+56)
	pdf.Line(121, y+41, 182, y+41)

	pdf.SetXY(20, y+13)
	pdf.Cell(10, 30, "Amount to pay:")
	pdf.SetXY(20, y+18)
	pdf.Cell(10, 30, "Reference:")
	pdf.SetXY(20, y+23)
	pdf.Cell(10, 30, "Due Date:")
	pdf.SetXY(20, y+28)
	pdf.Cell(10, 30, "IBAN:")
	pdf.SetXY(20, y+33)
	pdf.Cell(10, 30, "SWIFT:")
	pdf.SetXY(20, y+38)
	pdf.Cell(10, 30, "Bank Address:")

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(50, y+13)
	pdf.Cell(10, 30, c.Payment.Total+" "+c.Payment.Currency)
	pdf.SetXY(50, y+18)
	pdf.Cell(10, 30, c.Payment.ReferenceNumber)
	pdf.SetXY(50, y+23)
	pdf.Cell(10, 30, c.Payment.Due)
	pdf.SetXY(50, y+28)
	pdf.Cell(10, 30, c.Payment.Account)
	pdf.SetXY(50, y+33)
	pdf.Cell(10, 30, c.Payment.Swift)
	pdf.SetXY(50, y+38)
	pdf.Cell(10, 30, c.Payment.Bankaddress)

	pdf.SetXY(120, y+5)
	pdf.SetFont("Helvetica", "BI", 13)
	pdf.Cell(10, 30, "Detailed Amounts")

	pdf.SetFont("Helvetica", "I", 10)

	pdf.SetXY(120, y+13)
	pdf.Cell(10, 30, "Without VAT:")
	pdf.SetXY(120, y+18)
	pdf.Cell(10, 30, "VAT "+fmt.Sprintf("%d", c.Payment.VatProc)+" %:")
	pdf.SetXY(120, y+23)
	pdf.Cell(10, 30, "Total:")

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(160, y+13)
	pdf.Cell(10, 30, c.Payment.Amount+" "+c.Payment.Currency)
	pdf.SetXY(160, y+18)
	pdf.Cell(10, 30, c.Payment.Vat+" "+c.Payment.Currency)
	pdf.SetXY(160, y+23)
	pdf.Cell(10, 30, c.Payment.Total+" "+c.Payment.Currency)

	pdf.AliasNbPages("{nb}") // replace {nb}

	if strings.HasPrefix(c.Payment.Account, "FI") {
		pdf.SetXY(20, y+43)
		pdf.SetFont("Helvetica", "", 10)
		pdf.Cell(10, 30, "If you are in Finland, you can copypaste or scan following code:")
		pdf.SetXY(20, y+48)
		pdf.Cell(10, 30, c.Payment.Barcode)
		codeToFile(c.Payment.Barcode, "c.gif")
		var opt gofpdf.ImageOptions

		opt.ImageType = "gif"
		pdf.ImageOptions("c.gif", 20, y+68, 160, 20, false, opt, 0, "")
	}

	return pdf, nil
}
