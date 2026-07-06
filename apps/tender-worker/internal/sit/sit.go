// Package sit scrapes Singapore Institute of Technology's procurement-opportunities
// page. The list is a Drupal view: each tender is a light-grey card with an <h3>
// reference, a <p> title, two <time datetime> values (published, closing), and a
// "Find Out More" link to its detail page. Parsed with the standard library.
//
// SIT sits behind Imperva/Incapsula: a plain GET returns a JS-challenge stub
// (/_Incapsula_Resource) that must execute to set the ___utmvc clearance cookie.
// solveIncapsula runs that challenge in pure Go (see its TODO — pending the real
// challenge script). ponytail: brittle anti-bot; on failure Fetch errors → the
// source is logged + skipped (0 records), never crashing the worker.
package sit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/midtone/tender-worker/internal/store"
)

const (
	baseURL = "https://www.singaporetech.edu.sg"
	listURL = baseURL + "/procurement-opportunities"
)

var (
	cardSep = "layout--background--light-grey"
	h3Re    = regexp.MustCompile(`(?s)<h3>(.*?)</h3>`)
	titleRe = regexp.MustCompile(`(?s)<h3>.*?</h3>\s*<p>(.*?)</p>`)
	timeRe  = regexp.MustCompile(`datetime="([^"]+)"`)
	linkRe  = regexp.MustCompile(`href="(/procurement-opportunities/[^"#]+)"`)
	// The challenge stub loads the utmvc script from /_Incapsula_Resource?SWJIYLWA=…
	incapResRe = regexp.MustCompile(`src="(/_Incapsula_Resource\?[^"]*)"`)
)

// Fetch downloads the list page (clearing the Incapsula challenge first if present)
// and parses each tender card.
func Fetch(ctx context.Context) ([]store.Record, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	body, err := get(ctx, client, listURL)
	if err != nil {
		return nil, err
	}
	if isChallenge(body) {
		if err := solveIncapsula(ctx, client, body); err != nil {
			return nil, fmt.Errorf("sit: incapsula: %w", err)
		}
		if body, err = get(ctx, client, listURL); err != nil {
			return nil, err
		}
		if isChallenge(body) {
			return nil, fmt.Errorf("sit: still challenged after solve")
		}
	}
	return parse(body), nil
}

// parse extracts tender records from the Drupal view cards.
func parse(body []byte) []store.Record {
	cards := strings.Split(string(body), cardSep)
	var out []store.Record
	for _, c := range cards[1:] { // cards[0] is everything before the first card
		ref := strings.ToUpper(text(h3Re, c))
		if !strings.HasPrefix(ref, "TO") { // skip non-tender chunks
			continue
		}
		link := ""
		if m := linkRe.FindStringSubmatch(c); m != nil {
			link = baseURL + m[1]
		}
		dates := timeRe.FindAllStringSubmatch(c, -1)
		var published, closing *time.Time
		if len(dates) > 0 {
			published = parseTime(dates[0][1])
		}
		if len(dates) > 1 {
			closing = parseTime(dates[1][1])
		}
		out = append(out, store.Record{
			Source:        "sit",
			SourceID:      ref,
			ReferenceNo:   ref,
			Title:         text(titleRe, c),
			Link:          link,
			Agency:        "Singapore Institute of Technology",
			PublishedDate: published,
			ClosingDate:   closing,
		})
	}
	return out
}

// solveIncapsula clears the Imperva/Incapsula JS challenge so the reload returns
// real HTML: it fetches the /_Incapsula_Resource utmvc script the stub loads, runs
// it in goja under a faked browser env (browserEnvJS), harvests the ___utmvc cookie
// the script computes, and sets it in the client's jar.
//
// ponytail ceiling — KNOWN RISK: the utmvc script derives part of its fingerprint
// from browser surfaces (canvas/WebGL/font metrics) that a headless JS VM can't
// faithfully reproduce, so the digest may be junk and Incapsula may REJECT the
// cookie (the reload would re-challenge → Fetch errors → SIT skipped, 0 records).
// This was built + proven to *emit* a ___utmvc offline, but live acceptance is
// UNVERIFIED (the dev sandbox IP is Incapsula-blocked). Verify from the deploy host;
// if it re-challenges, the upgrade path is a headless browser. The env stubs below
// are the tuning surface.
func solveIncapsula(ctx context.Context, client *http.Client, body []byte) error {
	m := incapResRe.FindSubmatch(body)
	if m == nil {
		return fmt.Errorf("challenge resource not found")
	}
	js, err := get(ctx, client, baseURL+string(m[1]))
	if err != nil {
		return err
	}
	cookie, err := computeUtmvc(string(js))
	if err != nil {
		return err
	}
	u, _ := url.Parse(baseURL)
	client.Jar.SetCookies(u, []*http.Cookie{{
		Name:  "___utmvc",
		Value: strings.TrimPrefix(cookie, "___utmvc="),
		Path:  "/",
	}})
	return nil
}

// computeUtmvc runs the utmvc challenge script in goja under browserEnvJS and returns
// the "___utmvc=VALUE" cookie segment the script assigns to document.cookie.
func computeUtmvc(challengeJS string) (string, error) {
	vm := goja.New()
	var cookie string
	_ = vm.Set("__captureCookie", func(v string) {
		if cookie == "" && strings.HasPrefix(v, "___utmvc=") {
			cookie = strings.SplitN(v, ";", 2)[0]
		}
	})
	if _, err := vm.RunString(browserEnvJS); err != nil {
		return "", fmt.Errorf("env: %w", err)
	}
	// The challenge often throws internally (it probes APIs we don't fully fake) but
	// still sets the cookie first; ignore the error and check what was captured.
	_, _ = vm.RunString(challengeJS)
	if cookie == "" {
		return "", fmt.Errorf("challenge produced no ___utmvc cookie")
	}
	return cookie, nil
}

// browserEnvJS is a minimal faked browser environment for executing the Incapsula
// utmvc challenge in goja. document.cookie writes are forwarded to __captureCookie.
// This is the tuning surface for the ponytail ceiling noted on solveIncapsula.
const browserEnvJS = `
var window = this;
self = top = parent = window;
if(!Date.prototype.toGMTString){ Date.prototype.toGMTString = Date.prototype.toUTCString; }
(function(){
  var chars="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  window.btoa=function(input){var str=String(input),output="";for(var block,charCode,i=0,map=chars;str.charAt(i|0)||(map="=",i%1);output+=map.charAt(63&block>>8-i%1*8)){charCode=str.charCodeAt(i+=3/4);if(charCode>255)throw new Error("btoa range");block=block<<8|charCode;}return output;};
  window.atob=function(input){var str=String(input).replace(/=+$/,""),output="";if(str.length%4==1)throw new Error("atob");for(var bc=0,bs,buffer,i=0;buffer=str.charAt(i++);~buffer&&(bs=bc%4?bs*64+buffer:buffer,bc++%4)?output+=String.fromCharCode(255&bs>>(-2*bc&6)):0)buffer=chars.indexOf(buffer);return output;};
})();
atob = window.atob; btoa = window.btoa;
var navigator = {
  userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
  appName: "Netscape", appCodeName: "Mozilla", product: "Gecko", productSub: "20030107",
  appVersion: "5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
  platform: "Win32", vendor: "Google Inc.", vendorSub: "", language: "en-US",
  languages: ["en-US","en"], cookieEnabled: true, doNotTrack: null, onLine: true,
  hardwareConcurrency: 8, maxTouchPoints: 0, deviceMemory: 8,
  plugins: {length:0}, mimeTypes: {length:0}, javaEnabled: function(){return false;},
  userAgentData: {
    brands: [{brand:"Chromium",version:"124"},{brand:"Google Chrome",version:"124"},{brand:"Not-A.Brand",version:"99"}],
    mobile: false, platform: "Windows",
    getHighEntropyValues: function(hints){
      var v = {architecture:"x86", bitness:"64", model:"", platform:"Windows", platformVersion:"15.0.0",
               uaFullVersion:"124.0.0.0", fullVersionList:[{brand:"Chromium",version:"124.0.0.0"},{brand:"Google Chrome",version:"124.0.0.0"},{brand:"Not-A.Brand",version:"99.0.0.0"}], wow64:false};
      return { then:function(cb){ try{cb(v);}catch(e){} return this; }, catch:function(){ return this; } };
    }
  }
};
window.navigator = navigator;
window.history = { length: 2, pushState:function(){}, replaceState:function(){} };
window.performance = { now:function(){ return Date.now()%1e7; }, timing:{}, navigation:{type:0} };
window.innerWidth=1920; window.innerHeight=937; window.outerWidth=1920; window.outerHeight=1040;
window.screenX=0; window.screenY=0; window.screenLeft=0; window.screenTop=0;
window.pageXOffset=0; window.pageYOffset=0; window.devicePixelRatio=1;
var screen = { width:1920, height:1080, availWidth:1920, availHeight:1040, colorDepth:24, pixelDepth:24 };
window.screen = screen;
var location = {
  href: "https://www.singaporetech.edu.sg/procurement-opportunities",
  protocol: "https:", host: "www.singaporetech.edu.sg", hostname: "www.singaporetech.edu.sg",
  port: "", pathname: "/procurement-opportunities", search: "", hash: "",
  reload: function(){}, replace: function(){}, assign: function(){}, toString: function(){ return this.href; }
};
window.location = location;
function elem(tag){
  return { __tag:tag||"", style:{}, getAttribute:function(){return null;}, setAttribute:function(){},
    appendChild:function(c){return c;}, removeChild:function(c){return c;}, addEventListener:function(){},
    attachEvent:function(){}, cloneNode:function(){return elem(tag);}, submit:function(){}, click:function(){},
    setRequestHeader:function(){}, insertBefore:function(){}, childNodes:[], children:[], firstChild:null, parentNode:null,
    clientWidth:1920, clientHeight:937, offsetWidth:1920, offsetHeight:937, scrollWidth:1920, scrollHeight:937,
    width:300, height:150, innerHTML:"", innerText:"", value:"", href:"", src:"", action:"",
    getContext:function(){ return {
      fillRect:function(){}, clearRect:function(){}, fillText:function(){}, strokeText:function(){},
      beginPath:function(){}, closePath:function(){}, moveTo:function(){}, lineTo:function(){}, arc:function(){},
      fill:function(){}, stroke:function(){}, rect:function(){}, bezierCurveTo:function(){}, save:function(){}, restore:function(){},
      translate:function(){}, rotate:function(){}, scale:function(){}, isPointInPath:function(){return false;},
      measureText:function(t){ return { width: String(t).length*7.3251, actualBoundingBoxAscent:8, actualBoundingBoxDescent:2 }; },
      getImageData:function(x,y,w,h){ var n=((w||1)*(h||1)*4)|0, a=[]; for(var i=0;i<n;i++) a.push((i*131+7)&255); return {data:a,width:w,height:h}; },
      createLinearGradient:function(){ return {addColorStop:function(){}}; }, createRadialGradient:function(){ return {addColorStop:function(){}}; },
      getExtension:function(){ return { UNMASKED_VENDOR_WEBGL:37445, UNMASKED_RENDERER_WEBGL:37446 }; },
      getParameter:function(p){ if(p===37445) return "Google Inc. (NVIDIA)"; if(p===37446) return "ANGLE (NVIDIA GeForce GTX 1060)"; return ""; },
      getSupportedExtensions:function(){ return ["WEBGL_debug_renderer_info"]; },
      getShaderPrecisionFormat:function(){ return {precision:23, rangeMin:127, rangeMax:127}; },
      font:"", fillStyle:"", strokeStyle:"", textBaseline:"", textAlign:"", globalCompositeOperation:"", shadowColor:"", lineWidth:1
    }; },
    toDataURL:function(){ return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="; } };
}
var __cookiejar = {};
var document = {
  createElement: function(tag){ return elem(tag); }, createTextNode: function(){ return elem(); },
  getElementsByTagName: function(){ return [elem()]; }, getElementById: function(){ return elem(); },
  getElementsByClassName: function(){ return []; }, querySelector: function(){ return elem(); },
  querySelectorAll: function(){ return []; }, createEvent: function(){ return { initEvent:function(){} }; },
  documentElement: elem(), head: elem(), body: elem(), location: location, referrer: "",
  domain: "www.singaporetech.edu.sg", readyState: "complete", characterSet: "UTF-8"
};
window.document = document;
Object.defineProperty(document, "cookie", {
  get:function(){ var a=[]; for(var k in __cookiejar) a.push(k+"="+__cookiejar[k]); return a.join("; "); },
  set:function(v){ __captureCookie(v); var p=String(v).split(";")[0]; var i=p.indexOf("="); if(i>0){ __cookiejar[p.slice(0,i).replace(/^\s+|\s+$/g,"")]=p.slice(i+1); } }
});
window.setTimeout = function(f){ if(typeof f==="function"){ try{f();}catch(e){} } return 0; };
window.setInterval = function(){ return 0; }; window.clearTimeout = function(){}; window.clearInterval = function(){};
function __addEvent(t,f){ if(typeof f==="function" && (t==="load"||t==="DOMContentLoaded"||t==="readystatechange")){ try{f({type:t});}catch(e){} } }
window.addEventListener = __addEvent; document.addEventListener = __addEvent;
window.attachEvent = function(t,f){ __addEvent(t&&t.replace?t.replace(/^on/,""):t,f); }; document.attachEvent = window.attachEvent;
function XMLHttpRequest(){}
XMLHttpRequest.prototype.open=function(){}; XMLHttpRequest.prototype.send=function(){};
XMLHttpRequest.prototype.setRequestHeader=function(){}; XMLHttpRequest.prototype.getAllResponseHeaders=function(){return "";};
window.XMLHttpRequest = XMLHttpRequest;
`

// isChallenge reports whether body is the Incapsula JS-challenge / block stub.
func isChallenge(body []byte) bool {
	return bytes.Contains(body, []byte("_Incapsula_Resource"))
}

func parseTime(s string) *time.Time {
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(s)); err == nil {
		return &t
	}
	return nil
}

func text(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return store.StripHTML(m[1])
}

// get does a GET on the (cookie-jar) client with browser-like headers (Incapsula
// fingerprints requests) and returns the body (size-capped).
func get(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sit: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sit: %s status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}
