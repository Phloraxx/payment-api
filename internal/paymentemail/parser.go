package paymentemail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/sms"
)

const MaxRawBytes = 2 << 20

var (
	ErrUnrecognized   = errors.New("email is not a recognized bank UPI credit notification")
	ErrAmountMismatch = errors.New("email subject and body payment amounts do not agree")
	tagPattern        = regexp.MustCompile(`(?s)<[^>]*>`)
	spacePattern      = regexp.MustCompile(`\s+`)
)

type Message struct {
	MessageID             string
	From                  string
	Subject               string
	Body                  string
	Date                  time.Time
	AuthenticationResults []string
}

func ParseRaw(raw []byte) (Message, error) {
	if len(raw) == 0 {
		return Message{}, errors.New("raw email is required")
	}
	if len(raw) > MaxRawBytes {
		return Message{}, errors.New("raw email exceeds 2 MiB")
	}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return Message{}, err
	}
	from := decodeHeader(msg.Header.Get("From"))
	if parsed, err := mail.ParseAddress(from); err == nil {
		from = parsed.Address
	}
	subject := decodeHeader(msg.Header.Get("Subject"))
	body, err := readMIMEBody(msg.Header, msg.Body, 0)
	if err != nil {
		return Message{}, err
	}
	date, _ := mail.ParseDate(msg.Header.Get("Date"))
	return Message{
		MessageID:             strings.Trim(strings.TrimSpace(msg.Header.Get("Message-Id")), "<>"),
		From:                  strings.ToLower(strings.TrimSpace(from)),
		Subject:               strings.TrimSpace(subject),
		Body:                  normalizeText(body),
		Date:                  date.UTC(),
		AuthenticationResults: append([]string(nil), msg.Header["Authentication-Results"]...),
	}, nil
}

func Parse(message Message) (domain.ParsedSMS, error) {
	lowerSubject := strings.ToLower(message.Subject)
	lowerBody := strings.ToLower(message.Body)
	if !strings.Contains(lowerSubject, "slice account") ||
		(!strings.Contains(lowerSubject, "received") && !strings.Contains(lowerSubject, "credited")) {
		return domain.ParsedSMS{}, ErrUnrecognized
	}
	if !strings.Contains(lowerBody, "via upi") {
		return domain.ParsedSMS{}, ErrUnrecognized
	}
	subjectParsed, subjectErr := sms.Parse(message.Subject)
	bodyParsed, bodyErr := sms.Parse(message.Body)
	if errors.Is(subjectErr, sms.ErrUnrecognized) || errors.Is(bodyErr, sms.ErrUnrecognized) {
		return domain.ParsedSMS{}, ErrUnrecognized
	}
	if subjectErr != nil {
		return domain.ParsedSMS{}, subjectErr
	}
	if bodyErr != nil {
		return domain.ParsedSMS{}, bodyErr
	}
	if subjectParsed.AmountPaise != bodyParsed.AmountPaise {
		return domain.ParsedSMS{}, ErrAmountMismatch
	}
	bodyParsed.UPIId = strings.TrimRight(bodyParsed.UPIId, ".,;:")
	return bodyParsed, nil
}

func AuthenticatedSender(results []string, authServID, senderDomain string) bool {
	authServID = strings.ToLower(strings.TrimSpace(authServID))
	senderDomain = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(senderDomain), "@"))
	if authServID == "" || senderDomain == "" {
		return false
	}
	trustedResults := 0
	passed := false
	for _, raw := range results {
		value := strings.ToLower(spacePattern.ReplaceAllString(raw, " "))
		separator := strings.IndexByte(value, ';')
		if separator < 0 || strings.TrimSpace(value[:separator]) != authServID {
			continue
		}
		trustedResults++
		for _, method := range splitAuthenticationMethods(value[separator+1:]) {
			fields := strings.Fields(removeParentheticalComments(method))
			if len(fields) == 0 {
				continue
			}
			resultName, resultValue, ok := strings.Cut(fields[0], "=")
			if !ok || resultValue != "pass" {
				continue
			}
			properties := make(map[string]string, len(fields)-1)
			for _, field := range fields[1:] {
				name, propertyValue, ok := strings.Cut(strings.Trim(field, "();"), "=")
				if ok {
					properties[name] = strings.Trim(propertyValue, "<>\"")
				}
			}
			if resultName == "dkim" && (properties["header.d"] == senderDomain || strings.HasSuffix(properties["header.i"], "@"+senderDomain)) {
				passed = true
			}
			if resultName == "dmarc" && properties["header.from"] == senderDomain {
				passed = true
			}
		}
	}
	// Multiple results claiming the trusted receiver identity are ambiguous and
	// could include a forged pre-existing header, so authentication fails closed.
	return trustedResults == 1 && passed
}

func removeParentheticalComments(value string) string {
	var builder strings.Builder
	depth := 0
	for _, char := range value {
		switch char {
		case '(':
			depth++
			if depth == 1 {
				builder.WriteByte(' ')
			}
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				builder.WriteRune(char)
			}
		}
	}
	return builder.String()
}

func splitAuthenticationMethods(value string) []string {
	var methods []string
	start := 0
	for index := 0; index < len(value); index++ {
		if value[index] != ';' {
			continue
		}
		tail := strings.TrimLeft(value[index+1:], " \t")
		end := strings.IndexAny(tail, " \t;")
		if end < 0 {
			end = len(tail)
		}
		name, _, ok := strings.Cut(tail[:end], "=")
		if !ok || strings.Contains(name, ".") {
			continue
		}
		methods = append(methods, value[start:index])
		start = index + 1
	}
	methods = append(methods, value[start:])
	return methods
}

func readMIMEBody(header mail.Header, body io.Reader, depth int) (string, error) {
	if depth > 8 {
		return "", errors.New("email MIME nesting is too deep")
	}
	disposition, _, _ := mime.ParseMediaType(header.Get("Content-Disposition"))
	if strings.EqualFold(disposition, "attachment") {
		return "", nil
	}
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil || mediaType == "" {
		mediaType = "text/plain"
	}
	decoded := decodeTransfer(body, header.Get("Content-Transfer-Encoding"))
	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return "", errors.New("multipart email is missing a boundary")
		}
		reader := multipart.NewReader(decoded, boundary)
		var plain, fallback []string
		for {
			part, partErr := reader.NextPart()
			if errors.Is(partErr, io.EOF) {
				break
			}
			if partErr != nil {
				return "", partErr
			}
			partHeader := mail.Header(part.Header)
			text, partErr := readMIMEBody(partHeader, part, depth+1)
			if partErr != nil {
				return "", partErr
			}
			if strings.TrimSpace(text) == "" {
				continue
			}
			partType, _, _ := mime.ParseMediaType(partHeader.Get("Content-Type"))
			if partType == "text/plain" {
				plain = append(plain, text)
			} else {
				fallback = append(fallback, text)
			}
		}
		if len(plain) > 0 {
			return strings.Join(plain, "\n"), nil
		}
		return strings.Join(fallback, "\n"), nil
	}
	if mediaType != "text/plain" && mediaType != "text/html" {
		return "", nil
	}
	raw, err := io.ReadAll(io.LimitReader(decoded, MaxRawBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > MaxRawBytes {
		return "", errors.New("decoded email body exceeds 2 MiB")
	}
	text := string(raw)
	if mediaType == "text/html" {
		text = tagPattern.ReplaceAllString(text, " ")
		text = html.UnescapeString(text)
	}
	return text, nil
}

func decodeTransfer(body io.Reader, encoding string) io.Reader {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "quoted-printable":
		return quotedprintable.NewReader(body)
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, body)
	default:
		return body
	}
}

func decodeHeader(value string) string {
	decoded, err := new(mime.WordDecoder).DecodeHeader(value)
	if err != nil {
		return value
	}
	return decoded
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.TrimSpace(spacePattern.ReplaceAllString(value, " "))
	if utf8.RuneCountInString(value) > 64*1024 {
		runes := []rune(value)
		value = string(runes[:64*1024])
	}
	return value
}
