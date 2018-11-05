package script

const (
	OP_0       = 0x00 // 0
	OP_FALSE   = 0x00 // 0 - AKA OP_0
	OP_DATA_1  = 0x01 // 1
	OP_DATA_2  = 0x02 // 2
	OP_DATA_3  = 0x03 // 3
	OP_DATA_4  = 0x04 // 4
	OP_DATA_5  = 0x05 // 5
	OP_DATA_6  = 0x06 // 6
	OP_DATA_7  = 0x07 // 7
	OP_DATA_8  = 0x08 // 8
	OP_DATA_9  = 0x09 // 9
	OP_DATA_10 = 0x0a // 10
	OP_DATA_11 = 0x0b // 11
	OP_DATA_12 = 0x0c // 12
	OP_DATA_13 = 0x0d // 13
	OP_DATA_14 = 0x0e // 14
	OP_DATA_15 = 0x0f // 15
	OP_DATA_16 = 0x10 // 16
	OP_DATA_17 = 0x11 // 17
	OP_DATA_18 = 0x12 // 18
	OP_DATA_19 = 0x13 // 19
	OP_DATA_20 = 0x14 // 20
	OP_DATA_21 = 0x15 // 21
	OP_DATA_22 = 0x16 // 22
	OP_DATA_23 = 0x17 // 23
	OP_DATA_24 = 0x18 // 24
	OP_DATA_25 = 0x19 // 25
	OP_DATA_26 = 0x1a // 26
	OP_DATA_27 = 0x1b // 27
	OP_DATA_28 = 0x1c // 28
	OP_DATA_29 = 0x1d // 29
	OP_DATA_30 = 0x1e // 30
	OP_DATA_31 = 0x1f // 31
	OP_DATA_32 = 0x20 // 32
	OP_DATA_33 = 0x21 // 33
	OP_DATA_34 = 0x22 // 34
	OP_DATA_35 = 0x23 // 35
	OP_DATA_36 = 0x24 // 36
	OP_DATA_37 = 0x25 // 37
	OP_DATA_38 = 0x26 // 38
	OP_DATA_39 = 0x27 // 39
	OP_DATA_40 = 0x28 // 40
	OP_DATA_41 = 0x29 // 41
	OP_DATA_42 = 0x2a // 42
	OP_DATA_43 = 0x2b // 43
	OP_DATA_44 = 0x2c // 44
	OP_DATA_45 = 0x2d // 45
	OP_DATA_46 = 0x2e // 46
	OP_DATA_47 = 0x2f // 47
	OP_DATA_48 = 0x30 // 48
	OP_DATA_49 = 0x31 // 49
	OP_DATA_50 = 0x32 // 50
	OP_DATA_51 = 0x33 // 51
	OP_DATA_52 = 0x34 // 52
	OP_DATA_53 = 0x35 // 53
	OP_DATA_54 = 0x36 // 54
	OP_DATA_55 = 0x37 // 55
	OP_DATA_56 = 0x38 // 56
	OP_DATA_57 = 0x39 // 57
	OP_DATA_58 = 0x3a // 58
	OP_DATA_59 = 0x3b // 59
	OP_DATA_60 = 0x3c // 60
	OP_DATA_61 = 0x3d // 61
	OP_DATA_62 = 0x3e // 62
	OP_DATA_63 = 0x3f // 63
	OP_DATA_64 = 0x40 // 64
	OP_DATA_65 = 0x41 // 65
	OP_DATA_66 = 0x42 // 66
	OP_DATA_67 = 0x43 // 67
	OP_DATA_68 = 0x44 // 68
	OP_DATA_69 = 0x45 // 69
	OP_DATA_70 = 0x46 // 70
	OP_DATA_71 = 0x47 // 71
	OP_DATA_72 = 0x48 // 72
	OP_DATA_73 = 0x49 // 73
	OP_DATA_74 = 0x4a // 74
	OP_DATA_75 = 0x4b // 75

	OP_PUSHDATA1 = 0x4c // 76
	OP_PUSHDATA2 = 0x4d // 77
	OP_PUSHDATA4 = 0x4e // 78
	OP_1NEGATE   = 0x4f // 79
	OP_RESERVED  = 0x50 // 80

	OP_1    = 0x51 // 81 - AKA OP_TRUE
	OP_TRUE = 0x51 // 81
	OP_2    = 0x52 // 82
	OP_3    = 0x53 // 83
	OP_4    = 0x54 // 84
	OP_5    = 0x55 // 85
	OP_6    = 0x56 // 86
	OP_7    = 0x57 // 87
	OP_8    = 0x58 // 88
	OP_9    = 0x59 // 89
	OP_10   = 0x5a // 90
	OP_11   = 0x5b // 91
	OP_12   = 0x5c // 92
	OP_13   = 0x5d // 93
	OP_14   = 0x5e // 94
	OP_15   = 0x5f // 95
	OP_16   = 0x60 // 96

	OP_IF       = 0x63 // 99
	OP_NOTIF    = 0x64 // 100
	OP_VERIF    = 0x65 // 101
	OP_VERNOTIF = 0x66 // 102
	OP_ELSE     = 0x67 // 103
	OP_ENDIF    = 0x68 // 104
	OP_VERIFY   = 0x69 // 105

	OP_DUP = 0x76 // 118

	OP_CAT         = 0x7e // 126
	OP_SUBSTR      = 0x7f // 127
	OP_LEFT        = 0x80 // 128
	OP_RIGHT       = 0x81 // 129
	OP_SIZE        = 0x82 // 130
	OP_INVERT      = 0x83 // 131
	OP_AND         = 0x84 // 132
	OP_OR          = 0x85 // 133
	OP_XOR         = 0x86 // 134
	OP_EQUAL       = 0x87 // 135
	OP_EQUALVERIFY = 0x88 // 136
	OP_RESERVED1   = 0x89 // 137
	OP_RESERVED2   = 0x8a // 138
	OP_1ADD        = 0x8b // 139
	OP_1SUB        = 0x8c // 140
	OP_2MUL        = 0x8d // 141
	OP_2DIV        = 0x8e // 142
	OP_NEGATE      = 0x8f // 143
	OP_ABS         = 0x90 // 144
	OP_NOT         = 0x91 // 145
	OP_0NOTEQUAL   = 0x92 // 146
	OP_ADD         = 0x93 // 147
	OP_SUB         = 0x94 // 148
	OP_MUL         = 0x95 // 149
	OP_DIV         = 0x96 // 150
	OP_MOD         = 0x97 // 151
	OP_LSHIFT      = 0x98 // 152
	OP_RSHIFT      = 0x99 // 153

	OP_HASH160 = 0xa9 // 169
	OP_HASH256 = 0xaa // 170

	OP_CHECKSIG            = 0xac // 172
	OP_CHECKSIGVERIFY      = 0xad // 173
	OP_CHECKMULTISIG       = 0xae // 174
	OP_CHECKMULTISIGVERIFY = 0xaf // 175

	OP_CHECKLOCKTIMEVERIFY = 0xb1 // 177 - AKA OP_NOP2
	OP_CHECKSEQUENCEVERIFY = 0xb2 // 178 - AKA OP_NOP3
)

// OpcodeByName is a map that can be used to lookup an opcode by its
// human-readable name (OP_CHECKMULTISIG, OP_CHECKSIG, etc).
var OpcodeByName = make(map[string]byte)

func init() {
	// Initialize the opcode name to value map using the contents of the
	// opcode array.
	for _, op := range opcodeArray {
		OpcodeByName[op.name] = op.value
	}
	OpcodeByName["OP_FALSE"] = OP_FALSE
	OpcodeByName["OP_TRUE"] = OP_TRUE
	OpcodeByName["OP_CHECKLOCKTIMEVERIFY"] = OP_CHECKLOCKTIMEVERIFY
	OpcodeByName["OP_CHECKSEQUENCEVERIFY"] = OP_CHECKSEQUENCEVERIFY
}

// opcodeOnelineRepls defines opcode names which are replaced when doing a
// one-line disassembly.  This is done to match the output of the reference
// implementation while not changing the opcode names in the nicer full
// disassembly.
var opcodeOnelineRepls = map[string]string{
	"OP_1NEGATE": "-1",
	"OP_0":       "0",
	"OP_1":       "1",
	"OP_2":       "2",
	"OP_3":       "3",
	"OP_4":       "4",
	"OP_5":       "5",
	"OP_6":       "6",
	"OP_7":       "7",
	"OP_8":       "8",
	"OP_9":       "9",
	"OP_10":      "10",
	"OP_11":      "11",
	"OP_12":      "12",
	"OP_13":      "13",
	"OP_14":      "14",
	"OP_15":      "15",
	"OP_16":      "16",
}
