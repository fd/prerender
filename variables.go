package prerender

const (
	prerenderServiceURL = "http://service.prerender.io/"
	x_PRERENDER_TOKEN   = "X-Prerender-Token"
)

// googlebot, yahoo, and bingbot are not in this list because
// we support _escaped_fragment_ and want to ensure people aren't
// penalized for cloaking.
var crawlerUserAgents = []string{
	// 'googlebot',
	// 'yahoo',
	// 'bingbot',
	"baiduspider",
	"bufferbot",
	"developers.google.com/+/web/snippet",
	"embedly",
	"facebookexternalhit",
	"linkedinbot",
	"outbrain",
	"pinterest",
	"quora link preview",
	"rogerbot",
	"showyoubot",
	"slackbot",
	"twitterbot",
}

var extensionsToIgnore = []string{
	".ai",
	".avi",
	".css",
	".dat",
	".dmg",
	".doc",
	".doc",
	".exe",
	".flv",
	".gif",
	".ico",
	".iso",
	".jpeg",
	".jpg",
	".js",
	".less",
	".m4a",
	".m4v",
	".mov",
	".mp3",
	".mp4",
	".mpeg",
	".mpg",
	".pdf",
	".png",
	".ppt",
	".psd",
	".rar",
	".rss",
	".swf",
	".tif",
	".torrent",
	".txt",
	".wav",
	".wmv",
	".xls",
	".xml",
	".zip",
}
