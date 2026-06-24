package printer

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	esc = "\x1b"
	gs  = "\x1d"
)

func Init(w io.Writer) {
	w.Write([]byte(esc + "@"))
}

func AlignCenter(w io.Writer) {
	w.Write([]byte(esc + "a" + "\x01"))
}

func AlignLeft(w io.Writer) {
	w.Write([]byte(esc + "a" + "\x00"))
}

func AlignRight(w io.Writer) {
	w.Write([]byte(esc + "a" + "\x02"))
}

func BoldOn(w io.Writer) {
	w.Write([]byte(esc + "E" + "\x01"))
}

func BoldOff(w io.Writer) {
	w.Write([]byte(esc + "E" + "\x00"))
}

func DoubleSizeOn(w io.Writer) {
	w.Write([]byte(gs + "!" + "\x11"))
}

func DoubleSizeOff(w io.Writer) {
	w.Write([]byte(gs + "!" + "\x00"))
}

func Cut(w io.Writer) {
	w.Write([]byte(gs + "V" + "\x00"))
}

func Beep(w io.Writer) {
	w.Write([]byte(gs + "\x28" + "A" + "\x02" + "\x00" + "\x03" + "\x0a"))
}

func Line(w io.Writer) {
	w.Write([]byte(strings.Repeat("-", 42) + "\n"))
}

func Text(w io.Writer, s string) {
	w.Write([]byte(s + "\n"))
}

func TextBold(w io.Writer, s string) {
	BoldOn(w)
	w.Write([]byte(s + "\n"))
	BoldOff(w)
}

type TicketData struct {
	Negocio   string
	Direccion string
	Telefono  string
	Folio     int
	Fecha     string
	Cajero    string
	Items     []TicketItem
	Subtotal  float64
	Total     float64
	Pagos     []TicketPago
	Cambio    float64
}

type TicketItem struct {
	Nombre   string
	Cantidad float64
	Precio   float64
	Total    float64
}

type TicketPago struct {
	Metodo string
	Monto  float64
}

func metodoLabel(m string) string {
	switch m {
	case "e":
		return "Efectivo"
	case "t":
		return "Tarjeta"
	case "v":
		return "Vales"
	case "c":
		return "Credito"
	case "x":
		return "Transferencia"
	}
	return m
}

const pageWidth = 42

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func padLeft(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return strings.Repeat(" ", n-len(s)) + s
}

func PrintTicket(w io.Writer, td TicketData) {
	Init(w)

	AlignCenter(w)
	BoldOn(w)
	Text(w, td.Negocio)
	BoldOff(w)
	if td.Direccion != "" {
		Text(w, td.Direccion)
	}
	if td.Telefono != "" {
		Text(w, "Tel: "+td.Telefono)
	}
	Line(w)

	AlignLeft(w)
	Text(w, fmt.Sprintf("Folio: #%d", td.Folio))
	Text(w, fmt.Sprintf("Fecha: %s", td.Fecha))
	Text(w, fmt.Sprintf("Cajero: %s", td.Cajero))
	Line(w)

	AlignCenter(w)
	Text(w, "PRODUCTO      CANT  PRECIO   TOTAL")
	AlignLeft(w)

	for _, it := range td.Items {
		line := padRight(truncate(it.Nombre, 15), 15)
		line += padLeft(fmt.Sprintf("%.0f", it.Cantidad), 4)
		line += " "
		line += padLeft(fmt.Sprintf("$%.2f", it.Precio), 8)
		line += " "
		line += padLeft(fmt.Sprintf("$%.2f", it.Total), 8)
		Text(w, line)
	}

	Line(w)
	AlignRight(w)
	Text(w, fmt.Sprintf("Subtotal: $%.2f", td.Subtotal))
	BoldOn(w)
	DoubleSizeOn(w)
	Text(w, fmt.Sprintf("TOTAL: $%.2f", td.Total))
	DoubleSizeOff(w)
	BoldOff(w)

	Line(w)
	AlignLeft(w)
	Text(w, "Pagos:")
	for _, p := range td.Pagos {
		Text(w, fmt.Sprintf("  %s: $%.2f", metodoLabel(p.Metodo), p.Monto))
	}
	if td.Cambio > 0 {
		Text(w, fmt.Sprintf("Cambio: $%.2f", td.Cambio))
	}

	Line(w)
	AlignCenter(w)
	Text(w, "Gracias por su compra")
	Text(w, time.Now().Format("2006-01-02 15:04"))
	Text(w, "")

	Cut(w)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "."
}
