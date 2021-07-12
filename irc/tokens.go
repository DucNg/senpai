package irc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CasemapASCII of name is the canonical representation of name according to the
// ascii casemapping.
func CasemapASCII(name string) string {
	var sb strings.Builder
	sb.Grow(len(name))
	for _, r := range name {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// CasemapASCII of name is the canonical representation of name according to the
// rfc-1459 casemapping.
func CasemapRFC1459(name string) string {
	var sb strings.Builder
	sb.Grow(len(name))
	for _, r := range name {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		} else if r == '[' {
			r = '{'
		} else if r == ']' {
			r = '}'
		} else if r == '\\' {
			r = '|'
		} else if r == '~' {
			r = '^'
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// word returns the first word of s and the rest of s.
func word(s string) (word, rest string) {
	split := strings.SplitN(s, " ", 2)
	if len(split) < 2 {
		word = split[0]
		rest = ""
	} else {
		word = split[0]
		rest = split[1]
	}
	return
}

// tagEscape returns the the value of '\c' given c according to the message-tags
// specification.
func tagEscape(c rune) (escape rune) {
	switch c {
	case ':':
		escape = ';'
	case 's':
		escape = ' '
	case 'r':
		escape = '\r'
	case 'n':
		escape = '\n'
	default:
		escape = c
	}
	return
}

// unescapeTagValue removes escapes from the given string and replaces them with
// their meaningful values.
func unescapeTagValue(escaped string) string {
	var builder strings.Builder
	builder.Grow(len(escaped))
	escape := false

	for _, c := range escaped {
		if c == '\\' && !escape {
			escape = true
		} else {
			var cpp rune

			if escape {
				cpp = tagEscape(c)
			} else {
				cpp = c
			}

			builder.WriteRune(cpp)
			escape = false
		}
	}

	return builder.String()
}

// escapeTagValue does the inverse operation of unescapeTagValue.
func escapeTagValue(unescaped string) string {
	var sb strings.Builder
	sb.Grow(len(unescaped) * 2)

	for _, c := range unescaped {
		switch c {
		case ';':
			sb.WriteRune('\\')
			sb.WriteRune(':')
		case ' ':
			sb.WriteRune('\\')
			sb.WriteRune('s')
		case '\r':
			sb.WriteRune('\\')
			sb.WriteRune('r')
		case '\n':
			sb.WriteRune('\\')
			sb.WriteRune('n')
		case '\\':
			sb.WriteRune('\\')
			sb.WriteRune('\\')
		default:
			sb.WriteRune(c)
		}
	}

	return sb.String()
}

func parseTags(s string) (tags map[string]string) {
	s = s[1:]
	tags = map[string]string{}

	for _, item := range strings.Split(s, ";") {
		if item == "" || item == "=" || item == "+" || item == "+=" {
			continue
		}

		kv := strings.SplitN(item, "=", 2)
		if len(kv) < 2 {
			tags[kv[0]] = ""
		} else {
			tags[kv[0]] = unescapeTagValue(kv[1])
		}
	}

	return
}

var (
	errEmptyMessage      = errors.New("empty message")
	errIncompleteMessage = errors.New("message is incomplete")
)

var (
	errEmptyBatchID    = errors.New("empty BATCH ID")
	errNoPrefix        = errors.New("missing prefix")
	errNotEnoughParams = errors.New("not enough params")
	errUnknownCommand  = errors.New("unknown command")
)

type Prefix struct {
	Name string
	User string
	Host string
}

// ParsePrefix parses a "nick!user@host" combination (or a prefix) from the given
// string.
func ParsePrefix(s string) (p *Prefix) {
	if s == "" {
		return
	}

	p = &Prefix{}

	spl0 := strings.Split(s, "@")
	if 1 < len(spl0) {
		p.Host = spl0[1]
	}

	spl1 := strings.Split(spl0[0], "!")
	if 1 < len(spl1) {
		p.User = spl1[1]
	}

	p.Name = spl1[0]

	return
}

// Copy makes a copy of the prefix, but doesn't copy the internal strings.
func (p *Prefix) Copy() *Prefix {
	if p == nil {
		return nil
	}
	res := &Prefix{}
	*res = *p
	return res
}

// String returns the "nick!user@host" representation of the prefix.
func (p *Prefix) String() string {
	if p == nil {
		return ""
	}

	if p.User != "" && p.Host != "" {
		return p.Name + "!" + p.User + "@" + p.Host
	} else if p.User != "" {
		return p.Name + "!" + p.User
	} else if p.Host != "" {
		return p.Name + "@" + p.Host
	} else {
		return p.Name
	}
}

// Message is the representation of an IRC message.
type Message struct {
	Tags    map[string]string
	Prefix  *Prefix
	Command string
	Params  []string
}

func NewMessage(command string, params ...string) Message {
	return Message{Command: command, Params: params}
}

// ParseMessage parses the message from the given string, which must be trimmed
// of "\r\n" beforehand.
func ParseMessage(line string) (msg Message, err error) {
	line = strings.TrimLeft(line, " ")
	if line == "" {
		err = errEmptyMessage
		return
	}

	if line[0] == '@' {
		var tags string

		tags, line = word(line)
		msg.Tags = parseTags(tags)
	}

	line = strings.TrimLeft(line, " ")
	if line == "" {
		err = errIncompleteMessage
		return
	}

	if line[0] == ':' {
		var prefix string

		prefix, line = word(line)
		msg.Prefix = ParsePrefix(prefix[1:])
	}

	line = strings.TrimLeft(line, " ")
	if line == "" {
		err = errIncompleteMessage
		return
	}

	msg.Command, line = word(line)
	msg.Command = strings.ToUpper(msg.Command)

	msg.Params = make([]string, 0, 15)
	for line != "" {
		if line[0] == ':' {
			msg.Params = append(msg.Params, line[1:])
			break
		}

		var param string
		param, line = word(line)
		msg.Params = append(msg.Params, param)
	}

	return
}

func (msg Message) WithTag(key, value string) Message {
	if msg.Tags == nil {
		msg.Tags = map[string]string{}
	}
	msg.Tags[key] = escapeTagValue(value)
	return msg
}

// IsReply reports whether the message command is a server reply.
func (msg *Message) IsReply() bool {
	if len(msg.Command) != 3 {
		return false
	}
	for _, r := range msg.Command {
		if !('0' <= r && r <= '9') {
			return false
		}
	}
	return true
}

// String returns the protocol representation of the message, without an ending
// "\r\n".
func (msg *Message) String() string {
	var sb strings.Builder

	if msg.Tags != nil {
		sb.WriteRune('@')
		for k, v := range msg.Tags {
			sb.WriteString(k)
			if v != "" {
				sb.WriteRune('=')
				sb.WriteString(escapeTagValue(v))
			}
			sb.WriteRune(';')
		}
		sb.WriteRune(' ')
	}

	if msg.Prefix != nil {
		sb.WriteRune(':')
		sb.WriteString(msg.Prefix.String())
		sb.WriteRune(' ')
	}

	sb.WriteString(msg.Command)

	if len(msg.Params) != 0 {
		for _, p := range msg.Params[:len(msg.Params)-1] {
			sb.WriteRune(' ')
			sb.WriteString(p)
		}
		lastParam := msg.Params[len(msg.Params)-1]
		if !strings.ContainsRune(lastParam, ' ') && !strings.HasPrefix(lastParam, ":") {
			sb.WriteRune(' ')
			sb.WriteString(lastParam)
		} else {
			sb.WriteRune(' ')
			sb.WriteRune(':')
			sb.WriteString(lastParam)
		}
	}

	return sb.String()
}

// IsValid reports whether the message is correctly formed.
func (msg *Message) IsValid() bool {
	switch msg.Command {
	case "AUTHENTICATE", "PING", "PONG":
		return 1 <= len(msg.Params)
	case rplEndofnames, rplLoggedout, rplMotd, errNicknameinuse, rplNotopic, rplWelcome, rplYourhost:
		return 2 <= len(msg.Params)
	case rplIsupport, rplLoggedin, rplTopic, "FAIL", "WARN", "NOTE":
		return 3 <= len(msg.Params)
	case rplNamreply:
		return 4 <= len(msg.Params)
	case rplWhoreply:
		return 8 <= len(msg.Params)
	case "JOIN", "NICK", "PART", "TAGMSG":
		return 1 <= len(msg.Params) && msg.Prefix != nil
	case "KICK", "PRIVMSG", "NOTICE", "TOPIC":
		return 2 <= len(msg.Params) && msg.Prefix != nil
	case "QUIT":
		return msg.Prefix != nil
	case "CAP":
		return 3 <= len(msg.Params) &&
			(msg.Params[1] == "LS" ||
				msg.Params[1] == "LIST" ||
				msg.Params[1] == "ACK" ||
				msg.Params[1] == "NAK" ||
				msg.Params[1] == "NEW" ||
				msg.Params[1] == "DEL")
	case rplTopicwhotime:
		if len(msg.Params) < 4 {
			return false
		}
		_, err := strconv.ParseInt(msg.Params[3], 10, 64)
		return err == nil
	case "BATCH":
		if len(msg.Params) < 1 {
			return false
		}
		if len(msg.Params[0]) < 2 {
			return false
		}
		if msg.Params[0][0] == '+' {
			if len(msg.Params) < 2 {
				return false
			}
			switch msg.Params[1] {
			case "chathistory":
				return 3 <= len(msg.Params)
			default:
				return false
			}
		}
		return msg.Params[0][0] == '-'
	case "BOUNCER":
		return 3 <= len(msg.Params) && msg.Params[1] == "NETWORK"
	default:
		if len(msg.Command) != 3 || len(msg.Params) < 2 {
			return false
		}
		_, err := strconv.Atoi(msg.Command)
		return err == nil
	}
}

// Time returns the time when the message has been sent, if present.
func (msg *Message) Time() (t time.Time, ok bool) {
	var tag string
	var year, month, day, hour, minute, second, millis int

	tag, ok = msg.Tags["time"]
	if !ok {
		return
	}

	tag = strings.TrimSuffix(tag, "Z")

	_, err := fmt.Sscanf(tag, "%4d-%2d-%2dT%2d:%2d:%2d.%3d", &year, &month, &day, &hour, &minute, &second, &millis)
	if err != nil || month < 1 || 12 < month {
		ok = false
		return
	}

	t = time.Date(year, time.Month(month), day, hour, minute, second, millis*1e6, time.UTC)
	return
}

// TimeOrNow returns the time when the message has been sent, or time.Now() if
// absent.
func (msg *Message) TimeOrNow() time.Time {
	t, ok := msg.Time()
	if ok {
		return t
	}
	return time.Now().UTC()
}

// Severity is the severity of a server reply.
type Severity int

const (
	SeverityNote Severity = iota
	SeverityWarn
	SeverityFail
)

// ReplySeverity returns the severity of a server reply.
func ReplySeverity(reply string) Severity {
	switch reply[0] {
	case '4', '5':
		if reply == "422" {
			return SeverityNote
		} else {
			return SeverityFail
		}
	case '9':
		switch reply[2] {
		case '2', '4', '5', '6', '7':
			return SeverityFail
		default:
			return SeverityNote
		}
	default:
		return SeverityNote
	}
}

// Cap is a capability token in "CAP" server responses.
type Cap struct {
	Name   string
	Value  string
	Enable bool
}

// ParseCaps parses the last argument (capability list) of "CAP LS/LIST/NEW/DEL"
// server responses.
func ParseCaps(caps string) (diff []Cap) {
	for _, c := range strings.Split(caps, " ") {
		if c == "" || c == "-" || c == "=" || c == "-=" {
			continue
		}

		var item Cap

		if strings.HasPrefix(c, "-") {
			item.Enable = false
			c = c[1:]
		} else {
			item.Enable = true
		}

		kv := strings.SplitN(c, "=", 2)
		item.Name = strings.ToLower(kv[0])
		if len(kv) > 1 {
			item.Value = kv[1]
		}

		diff = append(diff, item)
	}

	return
}

// Member is a token in RPL_NAMREPLY's last parameter.
type Member struct {
	PowerLevel string
	Name       *Prefix
}

// ParseNameReply parses the last parameter of RPL_NAMREPLY, according to the
// membership prefixes of the server.
func ParseNameReply(trailing string, prefixes string) (names []Member) {
	for _, word := range strings.Split(trailing, " ") {
		if word == "" {
			continue
		}

		name := strings.TrimLeft(word, prefixes)
		names = append(names, Member{
			PowerLevel: word[:len(word)-len(name)],
			Name:       ParsePrefix(name),
		})
	}

	return
}
