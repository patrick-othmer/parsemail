package parsemail

// We add our tests in a separate file to prevent merge problems in case the original maintainer comes back.

import (
	"encoding/base64"
	"io/ioutil"
	"net/mail"
	"reflect"
	"strings"
	"testing"
	"time"
)

func Test_decodeMimeSentence(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"plain_ascii",
			args{
				`foo bar`,
			},
			`foo bar`,
		},
		{
			"utf_8_bmp",
			args{
				`=?utf-8?Q?F=C3=B8=C3=B8_bar?=`,
			},
			`Føø bar`,
		},
		{
			"utf_8_smp",
			args{
				`=?utf-8?Q?Cheers_=F0=9F=8D=BA!?=`,
			},
			`Cheers 🍺!`,
		},
		{
			"windows-1251",
			args{
				`=?windows-1251?Q?John_=C4oe?=`,
			},
			`John Дoe`,
		},
		{
			"windows-1252",
			args{
				`=?windows-1252?Q?John_Do=80?=`,
			},
			`John Do€`,
		},
		{
			"iso-8859-15",
			args{
				`=?iso-8859-15?Q?John_Do=A4?=`,
			},
			`John Do€`,
		},
		{
			"utf-7",
			args{
				`=?utf-7?B?Sm9obiBEbytJS3ct?=`,
			},
			`(removed text: non supported encoder)`,
		},
		{
			"gb2312",
			args{
				`=?gb2312?B?Sm9obiBEb2U=?=`,
			},
			`(removed text: non supported encoder)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeMimeSentence(tt.args.s); got != tt.want {
				t.Errorf("decodeMimeSentence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_headerParser_parseAddress(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		args   args
		wantMa *mail.Address
	}{
		{
			"plain_ascii",
			args{
				`test@example.com`,
			},
			&mail.Address{
				Address: `test@example.com`,
			},
		},
		{
			"utf_8_bmp",
			args{
				`=?utf-8?Q?John_D=C3=B8e?= <john.doe@example.com>`,
			},
			&mail.Address{
				Name:    `John Døe`,
				Address: `john.doe@example.com`,
			},
		},
		{
			"utf_8_smp",
			args{
				`=?utf-8?Q?John_=F0=9F=8D=BA_Doe?= <john.doe@example.com>`,
			},
			&mail.Address{
				Name:    `John 🍺 Doe`,
				Address: `john.doe@example.com`,
			},
		},
		{
			"windows-1251",
			args{
				`=?windows-1251?Q?John_=C4oe?= <john.doe@example.com>`,
			},
			&mail.Address{
				Name:    `John Дoe`,
				Address: `john.doe@example.com`,
			},
		},
		{
			"windows-1252",
			args{
				`=?windows-1252?Q?John_Do=80?= <john.doe@example.com>`,
			},
			&mail.Address{
				Name:    `John Do€`,
				Address: `john.doe@example.com`,
			},
		},
		{
			"iso-8859-15",
			args{
				`=?iso-8859-15?Q?John_Do=A4?= <john.doe@example.com>`,
			},
			&mail.Address{
				Name:    `John Do€`,
				Address: `john.doe@example.com`,
			},
		},
		{
			"utf-7",
			args{
				`=?utf-7?B?Sm9obiBEbytJS3ct?= <john.doe@example.com>`,
			},
			&mail.Address{
				Name:    `(removed text: non supported encoder)`,
				Address: `john.doe@example.com`,
			},
		},
		{
			"gb2312",
			args{
				`=?gb2312?B?Sm9obiBEb2U=?= <john.doe@example.com>`,
			},
			&mail.Address{
				Name:    `(removed text: non supported encoder)`,
				Address: `john.doe@example.com`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hp := headerParser{}
			if gotMa := hp.parseAddress(tt.args.s); !reflect.DeepEqual(gotMa, tt.wantMa) {
				t.Errorf("headerParser.parseAddress() = %v, want %v", gotMa, tt.wantMa)
			}
		})
	}
}

func Test_headerParser_parseAddressList(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		args   args
		wantMa []*mail.Address
	}{
		{
			"plain_ascii_single",
			args{
				`test@example.com`,
			},
			[]*mail.Address{
				{
					Address: `test@example.com`,
				},
			},
		},
		{
			"utf_8_bmp_single",
			args{
				`=?utf-8?Q?John_D=C3=B8e?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Name:    `John Døe`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"utf_8_smp_single",
			args{
				`=?utf-8?Q?John_=F0=9F=8D=BA_Doe?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Name:    `John 🍺 Doe`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"windows-1251",
			args{
				`=?windows-1251?Q?John_=C4oe?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Name:    `John Дoe`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"windows-1252",
			args{
				`=?windows-1252?Q?John_Do=80?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Name:    `John Do€`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"iso-8859-15",
			args{
				`=?iso-8859-15?Q?John_Do=A4?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Name:    `John Do€`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"utf-7",
			args{
				`=?utf-7?B?Sm9obiBEbytJS3ct?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Name:    `(removed text: non supported encoder)`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"gb2312",
			args{
				`=?gb2312?B?Sm9obiBEb2U=?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Name:    `(removed text: non supported encoder)`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"multiple_charsets with unsupported encoders",
			args{
				`test@example.com,=?utf-8?Q?John_D=C3=B8e?= <john.doe@example.com>,=?gb2312?B?Sm9obiBEb2U=?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Address: `test@example.com`,
				},
				{
					Name:    `John Døe`,
					Address: `john.doe@example.com`,
				},
				{
					Name:    `(removed text: non supported encoder)`,
					Address: `john.doe@example.com`,
				},
			},
		},
		{
			"multiple_charsets",
			args{
				`test@example.com,=?utf-8?Q?John_D=C3=B8e?= <john.doe@example.com>,=?windows-1251?Q?John_=C4oe?= <john.doe@example.com>`,
			},
			[]*mail.Address{
				{
					Address: `test@example.com`,
				},
				{
					Name:    `John Døe`,
					Address: `john.doe@example.com`,
				},
				{
					Name:    `John Дoe`,
					Address: `john.doe@example.com`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hp := headerParser{}
			if gotMa := hp.parseAddressList(tt.args.s); !reflect.DeepEqual(gotMa, tt.wantMa) {
				t.Errorf("headerParser.parseAddressList() = %v, want %v", gotMa, tt.wantMa)
			}
		})
	}
}

func TestParseEmail_on2it(t *testing.T) {
	var testData = map[string]struct {
		mailData string
		wantErr  bool

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
			mailData:    textPlainAttachmentInMultipart,
			contentType: `multipart/mixed; boundary="0000000000007e2bb40587e36196"`,
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
		"emptyPlaintextBase64Html": {
			mailData:    emptyPlaintextBase64Html,
			contentType: `multipart/alternative; boundary="--boundary_83159_42d3ef90-0a52-4a0c-9867-0ccf54ca8b80"`,
			subject:     "Some very important email",
			messageID:   "dshfkhhskjfdd0002eeaa@mail.example.org",
			from: []mail.Address{
				{
					Name:    "Example IT - Support",
					Address: "support@example.org",
				},
			},
			to: []mail.Address{
				{
					Name:    "Servicedesk",
					Address: "servicedesk@example.net",
				},
			},
			date:     parseDate("Sun, 07 Feb 2021 23:49:48 -0500"),
			textBody: ``,
			htmlBody: `<span>foo bar</span>`,
		},
		"evilShortContentDisposition": {
			mailData:    evilShortContentDisposition,
			wantErr:     true,
			contentType: `multipart/mixed; boundary="0000000000007e2bb40587e36196"`,
			subject:     "Evil hackerman panicking my mail parser",
			from: []mail.Address{
				{
					Address: "test@example.org",
				},
			},
			to: []mail.Address{
				{
					Address: "test@example.com",
				},
			},
			date:     parseDate("Thu, 02 May 2019 11:25:35 +0300"),
			textBody: `plain text part`,
		},
		"nestedMixed": {
			mailData:    nestedMixed,
			contentType: `multipart/mixed; boundary=--boundary_mixed_level_0`,
			subject:     "nested mixed mime multiparts",
			messageID:   "dshfkhhskjfdd0002eeaa@mail.example.org",
			from: []mail.Address{
				{
					Name:    "John Doe",
					Address: "john.doe@example.com",
				},
			},
			to: []mail.Address{
				{
					Name:    "",
					Address: "jane.doe@example.net",
				},
			},
			date:     parseDate("Sun, 07 Feb 2021 23:49:48 -0500"),
			textBody: `something something plain text`,
			htmlBody: `<span>something something html</span>`,
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
		if !td.wantErr {
			if err != nil {
				t.Error(err)
			}
		} else {
			if err == nil {
				t.Errorf("[Test Case %v] Expected error did not occur", index)
			}
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

var emptyPlaintextBase64Html = `Return-Path: <support@example.org>
Delivered-To: servicedesk@example.net
Received: from mail.example.org
	by mail.example.net (Dovecot) with LMTP id 7KTQOu3CIGCQiQAAhDWd3A
	for <servicedesk@example.net>; Mon, 08 Feb 2021 05:49:49 +0100
Received: from smtp.example.org (10.162.206.25) by
 mail.example.org (10.162.224.82) with Microsoft SMTP Server id
 15.1.1979.3 via Frontend Transport; Sun, 7 Feb 2021 23:49:48 -0500
Received: from somehost.example.org ([127.0.0.1]) by smtp.example.org with Microsoft SMTPSVC(8.5.9600.16384);
	 Sun, 7 Feb 2021 23:49:48 -0500
Importance: normal
Priority: normal
Content-Class: urn:content-classes:message
MIME-Version: 1.0
From: Example IT - Support <support@example.org>
To: Servicedesk <servicedesk@example.net>
Date: Sun, 7 Feb 2021 23:49:48 -0500
Subject: Some very important email
Content-Type: multipart/alternative;
	boundary="--boundary_83159_42d3ef90-0a52-4a0c-9867-0ccf54ca8b80"
Message-ID: <dshfkhhskjfdd0002eeaa@mail.example.org>
X-OriginalArrivalTime: 08 Feb 2021 04:49:48.0307 (UTC) FILETIME=[D4251630:01D6FDD5]

----boundary_83159_42d3ef90-0a52-4a0c-9867-0ccf54ca8b80
Content-Type: text/plain; charset="us-ascii"
Content-Transfer-Encoding: quoted-printable


----boundary_83159_42d3ef90-0a52-4a0c-9867-0ccf54ca8b80
Content-Type: text/html; charset="utf-8"
Content-Transfer-Encoding: base64

PHNwYW4+Zm9vIGJhcjwvc3Bhbj4=
----boundary_83159_42d3ef90-0a52-4a0c-9867-0ccf54ca8b80--
`

var evilShortContentDisposition = `From: test@example.org
Date: Thu, 2 May 2019 11:25:35 +0300
Subject: Evil hackerman panicking my mail parser
To: test@example.com
Content-Type: multipart/mixed; boundary="0000000000007e2bb40587e36196"

--0000000000007e2bb40587e36196
Content-Type: text/plain; charset="UTF-8"

plain text part
--0000000000007e2bb40587e36196
Content-Disposition: inline; xxx
Content-Type: application/octet-stream
Content-Transfer-Encoding: quoted-printable

attachment part
--0000000000007e2bb40587e36196--
`

var nestedMixed = `MIME-Version: 1.0
From: John Doe <john.doe@example.com>
To: jane.doe@example.net
Date: Sun, 7 Feb 2021 23:49:48 -0500
Subject: nested mixed mime multiparts
Content-Type: multipart/mixed;
 boundary=--boundary_mixed_level_0
Message-ID: <dshfkhhskjfdd0002eeaa@mail.example.org>

----boundary_mixed_level_0
Content-Type: multipart/alternative;
 boundary=--boundary_alternative_level_1

----boundary_alternative_level_1
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: quoted-printable

something something plain text

----boundary_alternative_level_1
Content-Type: text/html; charset=us-ascii
Content-Transfer-Encoding: quoted-printable

<span>something something html</span>
----boundary_alternative_level_1--

----boundary_mixed_level_0
Content-Type: multipart/mixed;
 boundary=--boundary_mixed_level_1

----boundary_mixed_level_1
Content-Disposition: attachment;
    filename=test.txt
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: quoted-printable

attachment text part
----boundary_mixed_level_1--

----boundary_mixed_level_0--
`
