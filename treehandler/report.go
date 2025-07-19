package treehandler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
)

func ReportHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	comment := r.FormValue("comment")
	clarity := r.FormValue("clarity")
	helpful := r.FormValue("helpful")
	unsafe := r.FormValue("unsafe")

	// Read file if uploaded
	var attachment []byte
	var filename string
	file, handler, err := r.FormFile("screenshot")
	if err == nil {
		defer file.Close()
		attachment, _ = io.ReadAll(file)
		filename = handler.Filename
	} else if err != http.ErrMissingFile {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	err = sendEmailWithAttachment(comment, clarity, helpful, unsafe, attachment, filename)
	if err != nil {
		http.Error(w, "Failed to send email: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "Report submitted successfully")
}

func sendEmailWithAttachment(comment, clarity, helpful, unsafe string, attachment []byte, filename string) error {
	from := "8088887230s@gmail.com"
	pass := "mzxh vzip snzs evls"
	to := []string{
		"8088887230s@gmail.com",
		"grvenu43@gmail.com",
	}

	subject := "Subject: Feedback Report with Attachment\r\n"
	boundary := "boundary123"

	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("From: %s\r\n", from))
	body.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ",")))
	body.WriteString(subject)
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	body.WriteString("\r\n--" + boundary + "\r\n")
	body.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")

	body.WriteString(fmt.Sprintf("Comment: %s\nClarity: %s\nHelpful: %s\nUnsafe: %s\n\n", comment, clarity, helpful, unsafe))

	if len(attachment) > 0 && filename != "" {
		body.WriteString("--" + boundary + "\r\n")
		body.WriteString(fmt.Sprintf("Content-Type: image/png\r\nContent-Disposition: attachment; filename=\"%s\"\r\n", filename))
		body.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")

		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment)))
		base64.StdEncoding.Encode(encoded, attachment)
		body.Write(encoded)
		body.WriteString("\r\n")
	}

	body.WriteString("--" + boundary + "--")

	auth := smtp.PlainAuth("", from, pass, "smtp.gmail.com")
	return smtp.SendMail("smtp.gmail.com:587", auth, from, to, body.Bytes())
}
