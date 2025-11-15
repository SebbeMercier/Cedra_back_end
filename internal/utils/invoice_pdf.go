package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/skip2/go-qrcode"
)

// GenerateSepaQR génère un QR SEPA (EPC) en base64 prêt à mettre dans <img src="...">
func GenerateSepaQR(iban, bic, name, ref string, amount float64) (string, error) {
	// format EPC basique
	sepa := fmt.Sprintf(`BCD
001
1
SCT
%s
%s
%s
EUR%.2f
%s`, bic, name, iban, amount, ref)

	png, err := qrcode.Encode(sepa, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// RenderReactInvoicePDF va charger ta page React/Next côté serveur et l’imprimer en PDF
// frontendURL doit ressembler à: http://localhost:3000/invoice
func RenderReactInvoicePDF(frontendURL, invoiceID, qrBase64 string) ([]byte, error) {
	// on passe les params en query
	q := url.Values{}
	q.Set("id", invoiceID)
	q.Set("qr", qrBase64)

	fullURL := fmt.Sprintf("%s?%s", frontendURL, q.Encode())

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// timeout pour éviter de bloquer
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var pdfBuf []byte

	err := chromedp.Run(ctx,
		chromedp.Navigate(fullURL),
		// on attend que le body soit là (ton composant CedraInvoice)
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Note: chromedp.PrintToPDF n'existe pas dans toutes les versions
			// Pour une solution simple, utilisez une bibliothèque PDF alternative
			// ou mettez à jour chromedp
			return fmt.Errorf("PrintToPDF non implémenté - utilisez une alternative comme wkhtmltopdf")
		}),
	)
	if err != nil {
		return nil, err
	}

	return pdfBuf, nil
}

// Helper: récupère l’URL du front (Next) depuis l'env
func GetFrontendInvoiceBaseURL() string {
	u := os.Getenv("FRONTEND_INVOICE_URL")
	if u == "" {
		// fallback local dev
		return "http://localhost:3000/invoice"
	}
	return u
}
