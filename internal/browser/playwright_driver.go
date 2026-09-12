package browser

import (
	"context"
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// PlaywrightDriver is the real browser automation implementation, using
// stock Playwright with no stealth/anti-detection configuration.
// CareerPilot does not attempt to evade platform bot detection
// (CLAUDE.md "CAPTCHA and anti-bot handling") — this driver reports what
// it observes and lets the caller decide what to do about it.
//
// Requires Playwright browser binaries to be installed on the host
// (playwright.Install()); this is an operational/runtime concern, not
// something this package manages implicitly.
type PlaywrightDriver struct {
	Headless bool

	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
}

// NewPlaywrightDriver returns a driver that launches Chromium.
func NewPlaywrightDriver(headless bool) *PlaywrightDriver {
	return &PlaywrightDriver{Headless: headless}
}

func (d *PlaywrightDriver) Launch(ctx context.Context) error {
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("browser: start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(d.Headless),
	})
	if err != nil {
		pw.Stop()
		return fmt.Errorf("browser: launch chromium: %w", err)
	}

	page, err := browser.NewPage()
	if err != nil {
		browser.Close()
		pw.Stop()
		return fmt.Errorf("browser: new page: %w", err)
	}

	d.pw = pw
	d.browser = browser
	d.page = page
	return nil
}

func (d *PlaywrightDriver) Navigate(ctx context.Context, url string) (PageState, error) {
	if _, err := d.page.Goto(url); err != nil {
		return StateNavigateErr, fmt.Errorf("browser: navigate to %s: %w", url, err)
	}
	return d.DetectState(ctx)
}

func (d *PlaywrightDriver) Fill(ctx context.Context, fields []FieldValue) error {
	for _, f := range fields {
		if err := d.page.Locator(f.Selector).Fill(f.Value); err != nil {
			return fmt.Errorf("browser: fill %s: %w", f.Selector, err)
		}
	}
	return nil
}

// captchaSignals and loginSignals are conservative, literal text/markup
// checks. This is detection only — it never attempts to solve or bypass
// what it finds (CLAUDE.md "Never implement: CAPTCHA solving/bypass").
var captchaSignals = []string{"g-recaptcha", "h-captcha", "cf-turnstile", "captcha"}
var loginSignals = []string{"sign in", "log in", "login"}

func (d *PlaywrightDriver) DetectState(ctx context.Context) (PageState, error) {
	content, err := d.page.Content()
	if err != nil {
		return StateUnknown, fmt.Errorf("browser: read page content: %w", err)
	}

	lower := strings.ToLower(content)
	for _, signal := range captchaSignals {
		if strings.Contains(lower, signal) {
			return StateCaptcha, nil
		}
	}

	// Login-wall detection is intentionally conservative: it only fires
	// when the page has essentially nothing else on it, avoiding false
	// positives on pages that merely contain a nav-bar login link.
	if len(strings.TrimSpace(content)) < 2000 {
		for _, signal := range loginSignals {
			if strings.Contains(lower, signal) {
				return StateLoginWall, nil
			}
		}
	}

	return StateNormal, nil
}

func (d *PlaywrightDriver) Screenshot(ctx context.Context) (Screenshot, error) {
	data, err := d.page.Screenshot(playwright.PageScreenshotOptions{
		Type: playwright.ScreenshotTypePng,
	})
	if err != nil {
		return Screenshot{}, fmt.Errorf("browser: screenshot: %w", err)
	}
	return Screenshot{Data: data, MimeType: "image/png"}, nil
}

func (d *PlaywrightDriver) CurrentURL(ctx context.Context) (string, error) {
	return d.page.URL(), nil
}

func (d *PlaywrightDriver) Submit(ctx context.Context, selector string) (PageState, error) {
	if err := d.page.Locator(selector).Click(); err != nil {
		return StateUnknown, fmt.Errorf("browser: click submit %s: %w", selector, err)
	}
	return d.DetectState(ctx)
}

func (d *PlaywrightDriver) Close(ctx context.Context) error {
	var errs []error
	if d.browser != nil {
		if err := d.browser.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if d.pw != nil {
		if err := d.pw.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("browser: close errors: %v", errs)
	}
	return nil
}
