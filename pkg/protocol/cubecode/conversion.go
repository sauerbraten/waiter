package cubecode

// conversion table: cubecode → unicode (i.e. using the cubecode code point as index)
// cubecode is a small subset of unicode containing selected characters from the Basic Latin, Latin-1 Supplement,
// Latin Extended-A and Cyrillic blocks, that can be represented in 8-bit space. characters included from the Basic
// Latin block (all characters except most control characters) keep their position in unicode. unused positions in
// the 8-bit space are filled up with letters from later Unicode blocks, resulting in interspersed Basic Latin and
// Latin-1 Supplement characters at the beginning of the conversion table.
// example: server sends a 2, cubeToUni[2] → Á
var cubeToUni = [256]rune{
	// Basic Latin (deliberately omitting most control characters)
	'\x00',
	// Latin-1 Supplement (selected letters)
	'À', 'Á', 'Â', 'Ã', 'Ä', 'Å', 'Æ',
	'Ç',
	// Basic Latin (cont.)
	'\t', '\n', '\v', '\f', '\r',
	// Latin-1 Supplement (cont.)
	'È', 'É', 'Ê', 'Ë',
	'Ì', 'Í', 'Î', 'Ï',
	'Ñ',
	'Ò', 'Ó', 'Ô', 'Õ', 'Ö', 'Ø',
	'Ù', 'Ú', 'Û',
	// Basic Latin (cont.)
	' ', '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	':', ';', '<', '=', '>', '?', '@',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'[', '\\', ']', '^', '_', '`',
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'{', '|', '}', '~',
	// Latin-1 Supplement (cont.)
	'Ü',
	'Ý',
	'ß',
	'à', 'á', 'â', 'ã', 'ä', 'å', 'æ',
	'ç',
	'è', 'é', 'ê', 'ë',
	'ì', 'í', 'î', 'ï',
	'ñ',
	'ò', 'ó', 'ô', 'õ', 'ö', 'ø',
	'ù', 'ú', 'û', 'ü',
	'ý', 'ÿ',
	// Latin Extended-A (selected letters)
	'Ą', 'ą',
	'Ć', 'ć', 'Č', 'č',
	'Ď', 'ď',
	'Ę', 'ę', 'Ě', 'ě',
	'Ğ', 'ğ',
	'İ', 'ı',
	'Ł', 'ł',
	'Ń', 'ń', 'Ň', 'ň',
	'Ő', 'ő', 'Œ', 'œ',
	'Ř', 'ř',
	'Ś', 'ś', 'Ş', 'ş', 'Š', 'š',
	'Ť', 'ť',
	'Ů', 'ů', 'Ű', 'ű',
	'Ÿ',
	'Ź', 'ź', 'Ż', 'ż', 'Ž', 'ž',
	// Cyrillic (selected letters, deliberately omitting letters visually identical to characters in Basic Latin)
	'Є',
	'Б' /**/, 'Г', 'Д', 'Ж', 'З', 'И', 'Й' /**/, 'Л' /*     */, 'П' /**/, 'У', 'Ф', 'Ц', 'Ч', 'Ш', 'Щ', 'Ъ', 'Ы', 'Ь', 'Э', 'Ю', 'Я',
	'б', 'в', 'г', 'д', 'ж', 'з', 'и', 'й', 'к', 'л', 'м', 'н', 'п', 'т' /**/, 'ф', 'ц', 'ч', 'ш', 'щ', 'ъ', 'ы', 'ь', 'э', 'ю', 'я',
	'є',
	'Ґ', 'ґ',
}

func ToUnicode(cpoint int32) rune {
	if !(-1 < cpoint && cpoint < 256) {
		return '�'
	}
	return cubeToUni[cpoint]
}

// conversion table: unicode → cubecode (i.e. using the unicode code point as key)
// reverse of cubeToUni.
// example: you want to send 'ø', uni2Cube['ø'] → 152, 152 should be encoded in the packet using PutInt().
var uniToCube = map[rune]int32{
	// Basic Latin (deliberately omitting most control characters)
	'\x00': 0,
	// Latin-1 Supplement (letters only)
	'À': 1, 'Á': 2, 'Â': 3, 'Ã': 4, 'Ä': 5, 'Å': 6, 'Æ': 7,
	'Ç': 8,
	// Basic Latin (cont.)
	'\t': 9, '\n': 10, '\v': 11, '\f': 12, '\r': 13,
	// Latin-1 Supplement (cont.)
	'È': 14, 'É': 15, 'Ê': 16, 'Ë': 17,
	'Ì': 18, 'Í': 19, 'Î': 20, 'Ï': 21,
	'Ñ': 22,
	'Ò': 23, 'Ó': 24, 'Ô': 25, 'Õ': 26, 'Ö': 27, 'Ø': 28,
	'Ù': 29, 'Ú': 30, 'Û': 31,
	// Basic Latin (cont.)
	' ': 32, '!': 33, '"': 34, '#': 35, '$': 36, '%': 37, '&': 38, '\'': 39, '(': 40, ')': 41, '*': 42, '+': 43, ',': 44, '-': 45, '.': 46, '/': 47,
	'0': 48, '1': 49, '2': 50, '3': 51, '4': 52, '5': 53, '6': 54, '7': 55, '8': 56, '9': 57,
	':': 58, ';': 59, '<': 60, '=': 61, '>': 62, '?': 63, '@': 64,
	'A': 65, 'B': 66, 'C': 67, 'D': 68, 'E': 69, 'F': 70, 'G': 71, 'H': 72, 'I': 73, 'J': 74, 'K': 75, 'L': 76, 'M': 77, 'N': 78, 'O': 79, 'P': 80, 'Q': 81, 'R': 82, 'S': 83, 'T': 84, 'U': 85, 'V': 86, 'W': 87, 'X': 88, 'Y': 89, 'Z': 90,
	'[': 91, '\\': 92, ']': 93, '^': 94, '_': 95, '`': 96,
	'a': 97, 'b': 98, 'c': 99, 'd': 100, 'e': 101, 'f': 102, 'g': 103, 'h': 104, 'i': 105, 'j': 106, 'k': 107, 'l': 108, 'm': 109, 'n': 110, 'o': 111, 'p': 112, 'q': 113, 'r': 114, 's': 115, 't': 116, 'u': 117, 'v': 118, 'w': 119, 'x': 120, 'y': 121, 'z': 122,
	'{': 123, '|': 124, '}': 125, '~': 126,
	// Latin-1 Supplement (cont.)
	'Ü': 127,
	'Ý': 128,
	'ß': 129,
	'à': 130, 'á': 131, 'â': 132, 'ã': 133, 'ä': 134, 'å': 135, 'æ': 136,
	'ç': 137,
	'è': 138, 'é': 139, 'ê': 140, 'ë': 141,
	'ì': 142, 'í': 143, 'î': 144, 'ï': 145,
	'ñ': 146,
	'ò': 147, 'ó': 148, 'ô': 149, 'õ': 150, 'ö': 151, 'ø': 152, 'ù': 153,
	'ú': 154, 'û': 155, 'ü': 156,
	'ý': 157, 'ÿ': 158,
	// Latin Extended-A (selected letters)
	'Ą': 159, 'ą': 160,
	'Ć': 161, 'ć': 162, 'Č': 163, 'č': 164,
	'Ď': 165, 'ď': 166,
	'Ę': 167, 'ę': 168, 'Ě': 169, 'ě': 170,
	'Ğ': 171, 'ğ': 172,
	'İ': 173, 'ı': 174,
	'Ł': 175, 'ł': 176,
	'Ń': 177, 'ń': 178, 'Ň': 179, 'ň': 180,
	'Ő': 181, 'ő': 182, 'Œ': 183, 'œ': 184,
	'Ř': 185, 'ř': 186,
	'Ś': 187, 'ś': 188, 'Ş': 189, 'ş': 190, 'Š': 191, 'š': 192,
	'Ť': 193, 'ť': 194,
	'Ů': 195, 'ů': 196, 'Ű': 197, 'ű': 198,
	'Ÿ': 199,
	'Ź': 200, 'ź': 201, 'Ż': 202, 'ż': 203, 'Ž': 204, 'ž': 205,
	// Cyrillic (selected letters, deliberately omitting letters visually identical to characters in Basic Latin)
	'Є': 206,
	'Б': 207 /*     */, 'Г': 208, 'Д': 209, 'Ж': 210, 'З': 211, 'И': 212, 'Й': 213 /*     */, 'Л': 214 /*              */, 'П': 215 /*     */, 'У': 216, 'Ф': 217, 'Ц': 218, 'Ч': 219, 'Ш': 220, 'Щ': 221, 'Ъ': 222, 'Ы': 223, 'Ь': 224, 'Э': 225, 'Ю': 226, 'Я': 227,
	'б': 228, 'в': 229, 'г': 230, 'д': 231, 'ж': 232, 'з': 233, 'и': 234, 'й': 235, 'к': 236, 'л': 37, 'м': 238, 'н': 239, 'п': 240, 'т': 241 /*     */, 'ф': 242, 'ц': 243, 'ч': 244, 'ш': 245, 'щ': 246, 'ъ': 247, 'ы': 248, 'ь': 249, 'э': 250, 'ю': 251, 'я': 252,
	'є': 253,
	'Ґ': 254, 'ґ': 255,
}

func FromUnicode(r rune) int32 {
	return uniToCube[r]
}
