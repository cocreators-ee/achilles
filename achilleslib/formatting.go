package achilleslib

import (
	"sync/atomic"

	"github.com/jeandeaual/go-locale"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var printer *message.Printer

func Int32Format(num int32) string {
	return printer.Sprintf("%v", number.Decimal(num))
}

func IntFormat(num int) string {
	return printer.Sprintf("%v", number.Decimal(num))
}

func AtomicIntFormat(num *atomic.Int32) string {
	return IntFormat(int(num.Load()))
}

func init() {
	userLanguage, err := locale.GetLanguage()
	if err != nil {
		userLanguage = "en"
	}

	printer = message.NewPrinter(language.Make(userLanguage))
}
