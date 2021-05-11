package parsemail

// We add our tests in a separate file to prevent merge problems in case the original mainainer comes back.

import (
	"encoding/base64"
	"io/ioutil"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestParseEmail_on2it(t *testing.T) {
	var testData = map[string]struct {
		mailData string

		contentType     string
		content         string
		subject         string
		date            time.Time
		from            []mail.Address
		sender          mail.Address
		to              []mail.Address
		replyTo         []mail.Address
		cc              []mail.Address
		bcc             []mail.Address
		messageID       string
		resentDate      time.Time
		resentFrom      []mail.Address
		resentSender    mail.Address
		resentTo        []mail.Address
		resentReplyTo   []mail.Address
		resentCc        []mail.Address
		resentBcc       []mail.Address
		resentMessageID string
		inReplyTo       []string
		references      []string
		htmlBody        string
		textBody        string
		attachments     []attachmentData
		embeddedFiles   []embeddedFileData
		headerCheck     func(mail.Header, *testing.T)
	}{
		"textPlainAttachmentInMultipart": {
			contentType: `multipart/mixed; boundary="0000000000007e2bb40587e36196"`,
			mailData:    textPlainAttachmentInMultipart,
			subject:     "Re: kern/54143 (virtualbox)",
			from: []mail.Address{
				{
					Name:    "Rares",
					Address: "rares@example.com",
				},
			},
			to: []mail.Address{
				{
					Name:    "",
					Address: "bugs@example.com",
				},
			},
			date:     parseDate("Fri, 02 May 2019 11:25:35 +0300"),
			textBody: `plain text part`,
			attachments: []attachmentData{
				{
					filename:    "test.txt",
					contentType: "text/plain",
					data:        "attachment text part",
				},
			},
		},
	}

	for index, td := range testData {
		e, err := Parse(strings.NewReader(td.mailData))
		if err != nil {
			t.Error(err)
		}

		if td.contentType != e.ContentType {
			t.Errorf("[Test Case %v] Wrong content type. Expected: %s, Got: %s", index, td.contentType, e.ContentType)
		}

		if td.content != "" {
			b, err := ioutil.ReadAll(e.Content)
			if err != nil {
				t.Error(err)
			} else if td.content != string(b) {
				t.Errorf("[Test Case %v] Wrong content. Expected: %s, Got: %s", index, td.content, string(b))
			}
		}

		if td.subject != e.Subject {
			t.Errorf("[Test Case %v] Wrong subject. Expected: %s, Got: %s", index, td.subject, e.Subject)
		}

		if td.messageID != e.MessageID {
			t.Errorf("[Test Case %v] Wrong messageID. Expected: '%s', Got: '%s'", index, td.messageID, e.MessageID)
		}

		if !td.date.Equal(e.Date) {
			t.Errorf("[Test Case %v] Wrong date. Expected: %v, Got: %v", index, td.date, e.Date)
		}

		d := dereferenceAddressList(e.From)
		if !assertAddressListEq(td.from, d) {
			t.Errorf("[Test Case %v] Wrong from. Expected: %s, Got: %s", index, td.from, d)
		}

		var sender mail.Address
		if e.Sender != nil {
			sender = *e.Sender
		}
		if td.sender != sender {
			t.Errorf("[Test Case %v] Wrong sender. Expected: %s, Got: %s", index, td.sender, sender)
		}

		d = dereferenceAddressList(e.To)
		if !assertAddressListEq(td.to, d) {
			t.Errorf("[Test Case %v] Wrong to. Expected: %s, Got: %s", index, td.to, d)
		}

		d = dereferenceAddressList(e.Cc)
		if !assertAddressListEq(td.cc, d) {
			t.Errorf("[Test Case %v] Wrong cc. Expected: %s, Got: %s", index, td.cc, d)
		}

		d = dereferenceAddressList(e.Bcc)
		if !assertAddressListEq(td.bcc, d) {
			t.Errorf("[Test Case %v] Wrong bcc. Expected: %s, Got: %s", index, td.bcc, d)
		}

		if td.resentMessageID != e.ResentMessageID {
			t.Errorf("[Test Case %v] Wrong resent messageID. Expected: '%s', Got: '%s'", index, td.resentMessageID, e.ResentMessageID)
		}

		if !td.resentDate.Equal(e.ResentDate) && !td.resentDate.IsZero() && !e.ResentDate.IsZero() {
			t.Errorf("[Test Case %v] Wrong resent date. Expected: %v, Got: %v", index, td.resentDate, e.ResentDate)
		}

		d = dereferenceAddressList(e.ResentFrom)
		if !assertAddressListEq(td.resentFrom, d) {
			t.Errorf("[Test Case %v] Wrong resent from. Expected: %s, Got: %s", index, td.resentFrom, d)
		}

		var resentSender mail.Address
		if e.ResentSender != nil {
			resentSender = *e.ResentSender
		}
		if td.resentSender != resentSender {
			t.Errorf("[Test Case %v] Wrong resent sender. Expected: %s, Got: %s", index, td.resentSender, resentSender)
		}

		d = dereferenceAddressList(e.ResentTo)
		if !assertAddressListEq(td.resentTo, d) {
			t.Errorf("[Test Case %v] Wrong resent to. Expected: %s, Got: %s", index, td.resentTo, d)
		}

		d = dereferenceAddressList(e.ResentCc)
		if !assertAddressListEq(td.resentCc, d) {
			t.Errorf("[Test Case %v] Wrong resent cc. Expected: %s, Got: %s", index, td.resentCc, d)
		}

		d = dereferenceAddressList(e.ResentBcc)
		if !assertAddressListEq(td.resentBcc, d) {
			t.Errorf("[Test Case %v] Wrong resent bcc. Expected: %s, Got: %s", index, td.resentBcc, d)
		}

		if !assertSliceEq(td.inReplyTo, e.InReplyTo) {
			t.Errorf("[Test Case %v] Wrong in reply to. Expected: %s, Got: %s", index, td.inReplyTo, e.InReplyTo)
		}

		if !assertSliceEq(td.references, e.References) {
			t.Errorf("[Test Case %v] Wrong references. Expected: %s, Got: %s", index, td.references, e.References)
		}

		d = dereferenceAddressList(e.ReplyTo)
		if !assertAddressListEq(td.replyTo, d) {
			t.Errorf("[Test Case %v] Wrong reply to. Expected: %s, Got: %s", index, td.replyTo, d)
		}

		if td.htmlBody != e.HTMLBody {
			t.Errorf("[Test Case %v] Wrong html body. Expected: '%s', Got: '%s'", index, td.htmlBody, e.HTMLBody)
		}

		if td.textBody != e.TextBody {
			t.Errorf("[Test Case %v] Wrong text body. Expected: '%s', Got: '%s'", index, td.textBody, e.TextBody)
		}

		if len(td.attachments) != len(e.Attachments) {
			t.Errorf("[Test Case %v] Incorrect number of attachments! Expected: %v, Got: %v.", index, len(td.attachments), len(e.Attachments))
		} else {
			attachs := e.Attachments[:]

			for _, ad := range td.attachments {
				found := false

				for i, ra := range attachs {
					b, err := ioutil.ReadAll(ra.Data)
					if err != nil {
						t.Error(err)
					}

					if ra.Filename == ad.filename && string(b) == ad.data && ra.ContentType == ad.contentType {
						found = true
						attachs = append(attachs[:i], attachs[i+1:]...)
					}
				}

				if !found {
					t.Errorf("[Test Case %v] Attachment not found: %s", index, ad.filename)
				}
			}

			if len(attachs) != 0 {
				t.Errorf("[Test Case %v] Email contains %v unexpected attachments: %v", index, len(attachs), attachs)
			}
		}

		if len(td.embeddedFiles) != len(e.EmbeddedFiles) {
			t.Errorf("[Test Case %v] Incorrect number of embedded files! Expected: %v, Got: %v.", index, len(td.embeddedFiles), len(e.EmbeddedFiles))
		} else {
			embeds := e.EmbeddedFiles[:]

			for _, ad := range td.embeddedFiles {
				found := false

				for i, ra := range embeds {
					b, err := ioutil.ReadAll(ra.Data)
					if err != nil {
						t.Error(err)
					}

					encoded := base64.StdEncoding.EncodeToString(b)

					if ra.CID == ad.cid && encoded == ad.base64data && ra.ContentType == ad.contentType {
						found = true
						embeds = append(embeds[:i], embeds[i+1:]...)
					}
				}

				if !found {
					t.Errorf("[Test Case %v] Embedded file not found: %s", index, ad.cid)
				}
			}

			if len(embeds) != 0 {
				t.Errorf("[Test Case %v] Email contains %v unexpected embedded files: %v", index, len(embeds), embeds)
			}
		}
	}
}

var textPlainAttachmentInMultipart = `From: Rares <rares@example.com>
Date: Thu, 2 May 2019 11:25:35 +0300
Subject: Re: kern/54143 (virtualbox)
To: bugs@example.com
Content-Type: multipart/mixed; boundary="0000000000007e2bb40587e36196"

--0000000000007e2bb40587e36196
Content-Type: text/plain; charset="UTF-8"

plain text part
--0000000000007e2bb40587e36196
Content-Disposition: attachment;
    filename=test.txt
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: quoted-printable

attachment text part
--0000000000007e2bb40587e36196--
`
