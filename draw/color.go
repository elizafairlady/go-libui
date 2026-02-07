package draw

// Standard colors (32-bit RGBA values).
const (
	DOpaque        = 0xFFFFFFFF
	DTransparent   = 0x00000000
	DBlack         = 0x000000FF
	DWhite         = 0xFFFFFFFF
	DRed           = 0xFF0000FF
	DGreen         = 0x00FF00FF
	DBlue          = 0x0000FFFF
	DCyan          = 0x00FFFFFF
	DMagenta       = 0xFF00FFFF
	DYellow        = 0xFFFF00FF
	DPaleyellow    = 0xFFFFAAFF
	DDarkyellow    = 0xEEEE9EFF
	DDarkgreen     = 0x448844FF
	DPalegreen     = 0xAAFFAAFF
	DMedgreen      = 0x88CC88FF
	DDarkblue      = 0x000055FF
	DPalebluegreen = 0xAAFFFFFF
	DPaleblue      = 0x0000BBFF
	DBluegreen     = 0x008888FF
	DGreygreen     = 0x55AAAAFF
	DPalegreygreen = 0x9EEEEEFF
	DYellowgreen   = 0x99994CFF
	DMedblue       = 0x000099FF
	DGreyblue      = 0x005DBBFF
	DPalegreyblue  = 0x4993DDFF
	DPurpleblue    = 0x8888CCFF

	// Acme-inspired UI colors
	DAcmeYellow  = 0xFFFFEAFF // warm cream body background
	DAcmeCyan    = 0xEAFFFFFF // pale cyan tag bar
	DAcmeGreen   = 0xE6FFE6FF // very pale green for success/ok
	DAcmeHigh    = 0xDDEEDDFF // muted green selection highlight
	DAcmeTag     = 0xEAFFFFFF // tag bar background
	DAcmeBorder  = 0x888888FF // subtle grey border
	DAcmeText    = 0x333333FF // soft black for text
	DAcmeDim     = 0x999999FF // dimmed/placeholder text
	DAcmeFocus   = 0x4488CCFF // calm blue for focus
	DAcmeButton  = 0xF0F0F0FF // light grey button background
	DAcmePressed = 0xDDDDDDFF // pressed button
	DAcmeInput   = 0xFFFFFEFF // near-white input background

	DNotacolor = 0xFFFFFF00
	DNofill    = DNotacolor
)
