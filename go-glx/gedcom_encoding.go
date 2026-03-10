// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package glx

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// convertToUTF8 detects the GEDCOM CHAR encoding from the header and converts
// the raw bytes to UTF-8 if needed. UTF-8, ASCII, and unknown encodings pass
// through unchanged.
func convertToUTF8(data []byte) []byte {
	charset := detectGEDCOMCharset(data)

	var enc encoding.Encoding

	switch strings.ToUpper(charset) {
	case "ANSI", "CP1252", "WINDOWS-1252":
		enc = charmap.Windows1252
	case "ANSEL":
		return convertANSELToUTF8(data)
	case "ISO-8859-1", "ISO8859-1", "LATIN1":
		enc = charmap.ISO8859_1
	default:
		// UTF-8, ASCII, empty, or unknown — pass through
		return data
	}

	result, _, err := transform.Bytes(enc.NewDecoder(), data)
	if err != nil {
		return data // fall back to raw bytes on error
	}

	return result
}

// detectGEDCOMCharset scans the first ~20 lines for "1 CHAR <value>" and
// returns the charset string. The CHAR line is always in the HEAD record near
// the top of the file, so a limited scan is sufficient.
func detectGEDCOMCharset(data []byte) string {
	// Scan up to 2KB or end of data for the CHAR line
	limit := 2048
	if len(data) < limit {
		limit = len(data)
	}

	chunk := string(data[:limit])

	for _, line := range strings.Split(chunk, "\n") {
		line = strings.TrimRight(line, "\r")
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "1" && strings.ToUpper(fields[1]) == "CHAR" {
			return fields[2]
		}
	}

	return ""
}

// anselToUTF8 maps ANSEL non-combining characters (0x80–0xBF range and select
// others) to their Unicode equivalents. Only the most common characters used
// in genealogical data are included.
var anselToUTF8 = map[byte]rune{
	0xA1: 0x0141, // Ł (L with stroke)
	0xA2: 0x00D8, // Ø (O with stroke)
	0xA3: 0x0110, // Đ (D with stroke)
	0xA4: 0x00DE, // Þ (Thorn)
	0xA5: 0x00C6, // Æ
	0xA6: 0x0152, // Œ
	0xA7: 0x02B9, // ʹ (soft sign)
	0xA8: 0x00B7, // · (middle dot)
	0xA9: 0x266D, // ♭ (flat)
	0xAA: 0x00AE, // ® (registered)
	0xAB: 0x00B1, // ± (plus-minus)
	0xAC: 0x01A0, // Ơ (O with horn)
	0xAD: 0x01AF, // Ư (U with horn)
	0xAE: 0x02BC, // ʼ (alif)
	0xB0: 0x02BB, // ʻ (ayn)
	0xB1: 0x0142, // ł (l with stroke)
	0xB2: 0x00F8, // ø (o with stroke)
	0xB3: 0x0111, // đ (d with stroke)
	0xB4: 0x00FE, // þ (thorn)
	0xB5: 0x00E6, // æ
	0xB6: 0x0153, // œ
	0xB7: 0x02BA, // ʺ (hard sign)
	0xB8: 0x0131, // ı (dotless i)
	0xB9: 0x00A3, // £ (pound)
	0xBA: 0x00F0, // ð (eth)
	0xBC: 0x01A1, // ơ (o with horn)
	0xBD: 0x01B0, // ư (u with horn)
	0xC0: 0x00B0, // ° (degree)
	0xC1: 0x2113, // ℓ (script l)
	0xC2: 0x00A9, // © (copyright)
	0xC3: 0x00A9, // © (copyright — ANSEL standard)
	0xC4: 0x266F, // ♯ (sharp)
	0xC5: 0x00BF, // ¿ (inverted question)
	0xC6: 0x00A1, // ¡ (inverted exclamation)
	0xCF: 0x00DF, // ß (eszett)
}

// anselCombining maps ANSEL combining diacritical marks (0xE0–0xFE) to Unicode
// combining characters. In ANSEL, the diacritical precedes the base letter;
// in Unicode, combining characters follow the base letter.
var anselCombining = map[byte]rune{
	0xE0: 0x0309, // combining hook above
	0xE1: 0x0300, // combining grave accent
	0xE2: 0x0301, // combining acute accent
	0xE3: 0x0302, // combining circumflex
	0xE4: 0x0303, // combining tilde
	0xE5: 0x0304, // combining macron
	0xE6: 0x0306, // combining breve
	0xE7: 0x0307, // combining dot above
	0xE8: 0x0308, // combining diaeresis (umlaut)
	0xE9: 0x030C, // combining caron (hacek)
	0xEA: 0x030A, // combining ring above
	0xEB: 0x0FE0, // combining ligature left half (approx)
	0xEC: 0x0FE1, // combining ligature right half (approx)
	0xED: 0x0315, // combining comma above right
	0xEE: 0x030B, // combining double acute
	0xEF: 0x0310, // combining candrabindu
	0xF0: 0x0327, // combining cedilla
	0xF1: 0x0328, // combining ogonek
	0xF2: 0x0323, // combining dot below
	0xF3: 0x0324, // combining diaeresis below
	0xF4: 0x0325, // combining ring below
	0xF5: 0x0333, // combining double low line
	0xF6: 0x0332, // combining low line
	0xF7: 0x0326, // combining comma below
	0xF8: 0x031C, // combining left half ring below
	0xF9: 0x032E, // combining breve below
	0xFE: 0x0313, // combining comma above
}

// convertANSELToUTF8 converts ANSEL-encoded bytes to UTF-8. ANSEL uses
// combining diacriticals (0xE0–0xFE) that precede the base letter, so the
// converter reorders them to follow the base letter (Unicode order).
func convertANSELToUTF8(data []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(data))

	i := 0
	for i < len(data) {
		b := data[i]

		// ASCII passthrough
		if b < 0x80 {
			buf.WriteByte(b)
			i++
			continue
		}

		// Combining diacritical (precedes base letter in ANSEL)
		if combiningRune, ok := anselCombining[b]; ok {
			// Next byte should be the base letter
			if i+1 < len(data) {
				base := data[i+1]
				if base < 0x80 {
					// Write base letter first, then combining mark (Unicode order)
					buf.WriteByte(base)
					var runeBytes [4]byte
					n := utf8.EncodeRune(runeBytes[:], combiningRune)
					buf.Write(runeBytes[:n])
					i += 2
					continue
				}
			}
			// No valid base letter follows — write the combining mark alone
			var runeBytes [4]byte
			n := utf8.EncodeRune(runeBytes[:], combiningRune)
			buf.Write(runeBytes[:n])
			i++
			continue
		}

		// Non-combining ANSEL character
		if r, ok := anselToUTF8[b]; ok {
			var runeBytes [4]byte
			n := utf8.EncodeRune(runeBytes[:], r)
			buf.Write(runeBytes[:n])
			i++
			continue
		}

		// Unknown high byte — replace with Unicode replacement character
		buf.WriteRune(utf8.RuneError)
		i++
	}

	return buf.Bytes()
}
