package assets

import _ "embed"

// TimelineCSS and TimelineJS draw timelines (a `type: timeline` page or a
// ```timeline fence): the snake, line and vertical layouts, era colours, and
// the fit that shrinks the text until every card has room.
//
//go:embed web/timeline.css
var TimelineCSS string

//go:embed web/timeline.js
var TimelineJS string
