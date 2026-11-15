package render

// igFlowIcons draw a flow box's icon (deck.IGFlowIcons, `[db] Orders`): line
// art on a 24 × 24 grid, stroked in the block's colour (.si-igf-ico).
var igFlowIcons = map[string]string{
	"user":     `<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4.4 3.6-8 8-8s8 3.6 8 8"/>`,
	"server":   `<rect x="3" y="3" width="18" height="7" rx="2"/><rect x="3" y="14" width="18" height="7" rx="2"/><path d="M7 6.5h.01M7 17.5h.01"/>`,
	"database": `<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/>`,
	"queue":    `<rect x="2" y="7" width="5" height="10" rx="1"/><rect x="9.5" y="7" width="5" height="10" rx="1"/><rect x="17" y="7" width="5" height="10" rx="1"/>`,
	"cache":    `<path d="M13 2 4 14h7l-1 8 9-12h-7z"/>`,
	"cloud":    `<path d="M7 19h10a5 5 0 0 0 .5-9.97A7 7 0 0 0 4.2 11.5 4 4 0 0 0 7 19z"/>`,
	"lock":     `<rect x="4" y="11" width="16" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/>`,
	"globe":    `<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.5 3.8 5.5 3.8 9s-1.3 6.5-3.8 9M12 3C9.5 5.5 8.2 8.5 8.2 12s1.3 6.5 3.8 9"/>`,
	"mobile":   `<rect x="6" y="2" width="12" height="20" rx="2.5"/><path d="M11 18h2"/>`,
	"function": `<path d="M9 21c2 0 2.6-1.6 3-4l2-10c.4-2.4 1.2-4 3.5-4M8 10h8"/>`,
	"browser":  `<rect x="2" y="4" width="20" height="16" rx="2"/><path d="M2 9h20M6 6.5h.01M9 6.5h.01"/>`,
	"mail":     `<rect x="2" y="5" width="20" height="14" rx="2"/><path d="m3 7 9 6 9-6"/>`,
	"file":     `<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>`,
	"api":      `<path d="M8 6 2 12l6 6M16 6l6 6-6 6M13.5 4l-3 16"/>`,
}
