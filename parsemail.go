package parsemail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	cs "golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

const contentTypeMultipartMixed = "multipart/mixed"
const contentTypeMultipartAlternative = "multipart/alternative"
const contentTypeMultipartAppleDouble = "multipart/appledouble"
const messageRFC822 = "message/rfc822"
const contentTypeMultipartSigned = "multipart/signed"
const contentTypeMultipartRelated = "multipart/related"
const contentTypeTextHtml = "text/html"
const contentTypeTextPlain = "text/plain"

// Parse an email message read from io.Reader into parsemail.Email struct
func Parse(r io.Reader) (email Email, err error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return
	}

	email, err = createEmailFromHeader(msg.Header)
	if err != nil {
		return
	}

	email.ContentType = msg.Header.Get("Content-Type")
	contentType, params, err := parseContentType(email.ContentType)
	if err != nil {
		return
	}

	encoding := strings.ToLower(msg.Header.Get("Content-Transfer-Encoding"))

	switch contentType {
	case contentTypeMultipartMixed, contentTypeMultipartSigned:
		email.TextBody, email.HTMLBody, email.Attachments, email.EmbeddedFiles, err = parseMultipartMixed(msg.Body, params["boundary"])
	case contentTypeMultipartAlternative:
		email.TextBody, email.HTMLBody, email.Attachments, email.EmbeddedFiles, err = parseMultipartAlternative(msg.Body, params["boundary"])
	case contentTypeMultipartRelated:
		email.TextBody, email.HTMLBody, email.Attachments, email.EmbeddedFiles, err = parseMultipartRelated(msg.Body, params["boundary"])
	case contentTypeTextPlain:
		var message []byte
		message, err = readAllDecode(msg.Body, encoding, email.ContentType)
		email.TextBody = strings.TrimSuffix(string(message[:]), "\n")
	case contentTypeTextHtml:
		var message []byte
		message, err = readAllDecode(msg.Body, encoding, email.ContentType)
		email.HTMLBody = strings.TrimSuffix(string(message[:]), "\n")
	default:
		email.Content, err = decodeContent(msg.Body, encoding)
	}

	return
}

func createEmailFromHeader(header mail.Header) (email Email, err error) {
	hp := headerParser{header: &header}

	email.Subject = decodeMimeSentence(header.Get("Subject"))
	email.From = hp.parseAddressList(header.Get("From"))
	email.Sender = hp.parseAddress(header.Get("Sender"))
	email.ReplyTo = hp.parseAddressList(header.Get("Reply-To"))
	email.To = hp.parseAddressList(header.Get("To"))
	email.Cc = hp.parseAddressList(header.Get("Cc"))
	email.Bcc = hp.parseAddressList(header.Get("Bcc"))
	email.Date = hp.parseTime(header.Get("Date"))
	email.ResentFrom = hp.parseAddressList(header.Get("Resent-From"))
	email.ResentSender = hp.parseAddress(header.Get("Resent-Sender"))
	email.ResentTo = hp.parseAddressList(header.Get("Resent-To"))
	email.ResentCc = hp.parseAddressList(header.Get("Resent-Cc"))
	email.ResentBcc = hp.parseAddressList(header.Get("Resent-Bcc"))
	email.ResentMessageID = hp.parseMessageId(header.Get("Resent-Message-ID"))
	email.MessageID = hp.parseMessageId(header.Get("Message-ID"))
	email.InReplyTo = hp.parseMessageIdList(header.Get("In-Reply-To"))
	email.References = hp.parseMessageIdList(header.Get("References"))
	email.ResentDate = hp.parseTime(header.Get("Resent-Date"))

	if hp.err != nil {
		err = hp.err
		return
	}

	//decode whole header for easier access to extra fields
	//todo: should we decode? aren't only standard fields mime encoded?
	email.Header, err = decodeHeaderMime(header)
	if err != nil {
		return
	}

	return
}

func parseContentType(contentTypeHeader string) (contentType string, params map[string]string, err error) {
	if contentTypeHeader == "" {
		contentType = contentTypeTextPlain
		return
	}

	return mime.ParseMediaType(contentTypeHeader)
}

func parseMultipartRelated(msg io.Reader, boundary string) (textBody, htmlBody string, attachments []Attachment, embeddedFiles []EmbeddedFile, err error) {
	pmr := multipart.NewReader(msg, boundary)
	for {
		part, err := pmr.NextPart()

		if err == io.EOF {
			break
		} else if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		contentType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		encoding := part.Header.Get("Content-Transfer-Encoding")

		switch contentType {
		case contentTypeTextPlain:
			ppContent, err := readAllDecode(part, encoding, part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			textBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		case contentTypeTextHtml:
			ppContent, err := readAllDecode(part, encoding, part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		case contentTypeMultipartMixed:
			tb, hb, at, ef, err := parseMultipartMixed(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += hb
			textBody += tb
			embeddedFiles = append(embeddedFiles, ef...)
			attachments = append(attachments, at...)
		case contentTypeMultipartAlternative:
			tb, hb, at, ef, err := parseMultipartAlternative(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += hb
			textBody += tb
			embeddedFiles = append(embeddedFiles, ef...)
			attachments = append(attachments, at...)
		default:
			if isEmbeddedFile(part) {
				ef, err := decodeEmbeddedFile(part)
				if err != nil {
					return textBody, htmlBody, attachments, embeddedFiles, err
				}

				embeddedFiles = append(embeddedFiles, ef)
			} else {
				return textBody, htmlBody, attachments, embeddedFiles, fmt.Errorf("Can't process multipart/related inner mime type: %s", contentType)
			}
		}
	}

	return textBody, htmlBody, attachments, embeddedFiles, err
}

func parseMultipartAlternative(msg io.Reader, boundary string) (textBody, htmlBody string, attachments []Attachment, embeddedFiles []EmbeddedFile, err error) {
	pmr := multipart.NewReader(msg, boundary)
	for {
		part, err := pmr.NextPart()

		if err == io.EOF {
			break
		} else if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		contentType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		encoding := part.Header.Get("Content-Transfer-Encoding")

		switch contentType {
		case contentTypeTextPlain:
			ppContent, err := readAllDecode(part, encoding, part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			textBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		case contentTypeTextHtml:
			ppContent, err := readAllDecode(part, encoding, part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		case contentTypeMultipartRelated:
			tb, hb, at, ef, err := parseMultipartRelated(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += hb
			textBody += tb
			embeddedFiles = append(embeddedFiles, ef...)
			attachments = append(attachments, at...)
		case contentTypeMultipartMixed:
			tb, hb, at, ef, err := parseMultipartMixed(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += hb
			textBody += tb
			embeddedFiles = append(embeddedFiles, ef...)
			attachments = append(attachments, at...)
		default:
			if isEmbeddedFile(part) {
				ef, err := decodeEmbeddedFile(part)
				if err != nil {
					return textBody, htmlBody, attachments, embeddedFiles, err
				}

				embeddedFiles = append(embeddedFiles, ef)
			} else {
				return textBody, htmlBody, attachments, embeddedFiles, fmt.Errorf("Can't process multipart/alternative inner mime type: %s", contentType)
			}
		}
	}

	return textBody, htmlBody, attachments, embeddedFiles, err
}

func parseMultipartMixed(msg io.Reader, boundary string) (textBody, htmlBody string, attachments []Attachment, embeddedFiles []EmbeddedFile, err error) {
	mr := multipart.NewReader(msg, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		contentType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		if isAttachment(part) {
			at, err := decodeAttachment(part)
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}
			attachments = append(attachments, at)
			continue
		}

		encoding := part.Header.Get("Content-Transfer-Encoding")

		if contentType == contentTypeMultipartAlternative {
			textBody, htmlBody, attachments, embeddedFiles, err = parseMultipartAlternative(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}
		} else if contentType == contentTypeMultipartMixed {
			tb, hb, at, ef, err := parseMultipartMixed(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += hb
			textBody += tb
			embeddedFiles = append(embeddedFiles, ef...)
			attachments = append(attachments, at...)
		} else if contentType == contentTypeMultipartRelated {
			textBody, htmlBody, attachments, embeddedFiles, err = parseMultipartRelated(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}
		} else if contentType == contentTypeTextPlain {
			ppContent, err := readAllDecode(part, encoding, part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			textBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		} else if contentType == contentTypeTextHtml {
			ppContent, err := readAllDecode(part, encoding, part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		} else if isEmbeddedFile(part) {
			ef, err := decodeEmbeddedFile(part)
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			embeddedFiles = append(embeddedFiles, ef)
		} else {
			return textBody, htmlBody, attachments, embeddedFiles, fmt.Errorf("Unknown multipart/mixed nested mime type: %s", contentType)
		}
	}

	return textBody, htmlBody, attachments, embeddedFiles, err
}

func decodeMimeSentence(s string) string {
	result := []string{}
	ss := strings.Split(s, " ")

	for _, word := range ss {
		word = removeUnsupportedEncoding(word)

		w, err := mimeWordDecoder.Decode(word)
		if err != nil {
			if len(result) == 0 {
				w = word
			} else {
				w = " " + word
			}
		}

		result = append(result, w)
	}

	return strings.Join(result, "")
}

func removeUnsupportedEncodingForAddress(s string) string {
	if s == "" {
		return s
	}

	ss := strings.Split(s, " ")
	result := []string{}

	for _, word := range ss {
		validWord := word

		if !(strings.HasPrefix(word, "=?") && strings.HasSuffix(word, "?=")) {
			result = append(result, validWord)

			continue
		}

		word = word[2 : len(word)-2]

		// split word "UTF-8?q?text" into "UTF-8", 'q', and "text"
		charset, text, _ := strings.Cut(word, "?")
		if charset == "" {
			validWord = `"(removed text: non supported charset)"`
		}

		encoding, _, _ := strings.Cut(text, "?")
		if len(encoding) != 1 {
			validWord = `"(removed text: non supported encoding)"`
		}

		if charset != "" {
			encoder, _ := ianaindex.MIME.Encoding(charset)

			if encoder == nil {
				validWord = `"(removed text: non supported encoder)"`
			}
		}

		result = append(result, validWord)
	}

	return strings.Join(result, " ")
}

func removeUnsupportedEncodingForAddressList(s string) string {
	if s == "" {
		return s
	}

	addresses := s
	result := []string{}

	for _, address := range strings.Split(addresses, ",") {
		result = append(result, removeUnsupportedEncodingForAddress(address))
	}

	return strings.Join(result, ",")
}

func removeUnsupportedEncoding(s string) string {
	if s == "" {
		return s
	}

	word := s

	if !(strings.HasPrefix(word, "=?") && strings.HasSuffix(word, "?=")) {
		return word
	}

	word = word[2 : len(word)-2]

	// split word "UTF-8?q?text" into "UTF-8", 'q', and "text"
	charset, text, _ := strings.Cut(word, "?")
	if charset == "" {
		return "(removed text: non supported charset)"
	}

	encoding, _, _ := strings.Cut(text, "?")
	if len(encoding) != 1 {
		return "(removed text: non supported encoding)"
	}

	if charset != "" {
		encoder, _ := ianaindex.MIME.Encoding(charset)

		if encoder == nil {
			return "(removed text: non supported encoder)"
		}
	}

	return s
}

func decodeHeaderMime(header mail.Header) (mail.Header, error) {
	parsedHeader := map[string][]string{}

	for headerName, headerData := range header {

		parsedHeaderData := []string{}
		for _, headerValue := range headerData {
			parsedHeaderData = append(parsedHeaderData, decodeMimeSentence(headerValue))
		}

		parsedHeader[headerName] = parsedHeaderData
	}

	return mail.Header(parsedHeader), nil
}

func isEmbeddedFile(part *multipart.Part) bool {
	return part.Header.Get("Content-Transfer-Encoding") != "" || strings.HasPrefix(part.Header.Get("Content-Disposition"), "inline; filename=")
}

func decodeEmbeddedFile(part *multipart.Part) (ef EmbeddedFile, err error) {
	cid := decodeMimeSentence(part.Header.Get("Content-Id"))
	decoded, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"))
	if err != nil {
		return
	}

	ef.CID = strings.Trim(cid, "<>")
	if ef.CID == "" {
		_, param, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil {
			return ef, err
		}

		if _, ok := param["filename"]; ok {
			ef.CID = param["filename"]
		}
	}

	ef.Data = decoded

	contentType := part.Header.Get("Content-Type")
	if strings.Contains(contentType, ";") {
		contentType = strings.SplitN(contentType, ";", 2)[0]
	}
	ef.ContentType = contentType

	return
}

// Everything that is not html or plain is treated as an attachment.
func isAttachment(part *multipart.Part) bool {
	if part.Header.Get("Content-Disposition") != "" {
		contentDisposition, _, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil {
			return false
		}

		if contentDisposition == "attachment" {
			return true
		}
	}

	return false
}

func decodeAttachment(part *multipart.Part) (at Attachment, err error) {
	filename := ""
	if part.Header.Get("Content-Type") == messageRFC822 {
		filename = strings.Trim(decodeMimeSentence(part.Header.Get("Content-Id")), "<>") + ".eml"
	} else {
		filename = decodeMimeSentence(part.FileName())
	}

	if part.Header.Get("Content-Type") == messageRFC822 {
		dd, err := ioutil.ReadAll(part)
		if err != nil {
			return at, err
		}
		at.Data = bytes.NewReader(dd)
	} else {
		at.Data, err = decodeContent(part, part.Header.Get("Content-Transfer-Encoding"))
		if err != nil {
			return
		}
	}

	at.Filename = filename
	at.ContentType = strings.Split(part.Header.Get("Content-Type"), ";")[0]

	return
}

func readAllDecode(content io.Reader, encoding, contentType string) ([]byte, error) {
	r, err := decodeContent(content, encoding)
	if err != nil {
		return nil, err
	}

	cr, err := cs.NewReader(r, contentType)
	if err == io.EOF {
		return []byte{}, nil
	} else if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(cr)
}

func decodeContent(content io.Reader, encoding string) (io.Reader, error) {
	encoding = strings.ToLower(encoding)

	switch encoding {
	case "base64":
		decoded := base64.NewDecoder(base64.StdEncoding, content)
		b, err := ioutil.ReadAll(decoded)
		if err != nil {
			return nil, err
		}

		return bytes.NewReader(b), nil
	case "7bit", "", "8bit":
		dd, err := ioutil.ReadAll(content)
		if err != nil {
			return nil, err
		}

		return bytes.NewReader(dd), nil
	case "quoted-printable":
		decoded := quotedprintable.NewReader(content)
		b, err := ioutil.ReadAll(decoded)
		if err != nil {
			return nil, err
		}

		return bytes.NewReader(b), nil
	default:
		return nil, fmt.Errorf("unknown encoding: %s", encoding)
	}
}

type headerParser struct {
	header *mail.Header
	err    error
}

// This is needed because the default address parser only understands utf-8, iso-8859-1, and us-ascii.
var mimeWordDecoder = &mime.WordDecoder{
	CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
		enc, err := ianaindex.MIME.Encoding(charset)
		if err != nil {
			return nil, err
		}

		if enc == nil {
			return nil, fmt.Errorf("invalid encoding for charset %s", charset)
		}

		return transform.NewReader(input, enc.NewDecoder()), nil
	},
}

var addressParser = mail.AddressParser{
	WordDecoder: mimeWordDecoder,
}

func (hp headerParser) parseAddress(s string) (ma *mail.Address) {
	if hp.err != nil {
		return nil
	}

	if strings.Trim(s, " \n") != "" {
		ma, hp.err = addressParser.Parse(removeUnsupportedEncodingForAddress(s))

		return ma
	}

	return nil
}

func (hp headerParser) parseAddressList(s string) (ma []*mail.Address) {
	if hp.err != nil {
		return
	}

	if strings.Trim(s, " \n") != "" {
		ma, hp.err = addressParser.ParseList(removeUnsupportedEncodingForAddressList(s))
		return
	}

	return
}

func (hp headerParser) parseTime(s string) (t time.Time) {
	if hp.err != nil || s == "" {
		return
	}

	formats := []string{
		time.RFC1123Z,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		time.RFC1123Z + " (MST)",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
	}

	for _, format := range formats {
		t, hp.err = time.Parse(format, s)
		if hp.err == nil {
			return
		}
	}

	return
}

func (hp headerParser) parseMessageId(s string) string {
	if hp.err != nil {
		return ""
	}

	return strings.Trim(s, "<> ")
}

func (hp headerParser) parseMessageIdList(s string) (result []string) {
	if hp.err != nil {
		return
	}

	for _, p := range strings.Split(s, " ") {
		if strings.Trim(p, " \n") != "" {
			result = append(result, hp.parseMessageId(p))
		}
	}

	return
}

// Attachment with filename, content type and data (as a io.Reader)
type Attachment struct {
	Filename    string
	ContentType string
	Data        io.Reader
}

// EmbeddedFile with content id, content type and data (as a io.Reader)
type EmbeddedFile struct {
	CID         string
	ContentType string
	Data        io.Reader
}

// Email with fields for all the headers defined in RFC5322 with it's attachments and
type Email struct {
	Header mail.Header

	Subject    string
	Sender     *mail.Address
	From       []*mail.Address
	ReplyTo    []*mail.Address
	To         []*mail.Address
	Cc         []*mail.Address
	Bcc        []*mail.Address
	Date       time.Time
	MessageID  string
	InReplyTo  []string
	References []string

	ResentFrom      []*mail.Address
	ResentSender    *mail.Address
	ResentTo        []*mail.Address
	ResentDate      time.Time
	ResentCc        []*mail.Address
	ResentBcc       []*mail.Address
	ResentMessageID string

	ContentType string
	Content     io.Reader

	HTMLBody string
	TextBody string

	Attachments   []Attachment
	EmbeddedFiles []EmbeddedFile
}
