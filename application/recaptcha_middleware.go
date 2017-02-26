// Package application contains the server-side application code.
package application

/*
 * The MIT License (MIT)
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
import (
	"github.com/rvelhote/go-recaptcha"
	"github.com/gorilla/securecookie"
	"net/http"
	"time"
	"context"
	"net"
)

// cookieName specifies the cookie name that will hold the verification of wether the reCAPTCHA passed or not.
//const cookieName = "reCAPTCHA"

// Cookie is a single instance of the reCAPTCHA cookie to avoid instantiation
// FIXME We will want to set an expire date so clearly this won't work :P
//var Cookie = &http.Cookie{Name: cookieName, Value: "1", HttpOnly: true, Path: "/"}

// RecaptchaMiddleware will allow us to chain the reCAPTCHA validation into the request processing
type RecaptchaMiddleware struct {
	// Configuration holds the application configuration. In this context only the reCAPTCHA configuration is used
	Configuration Configuration

	//
	SecureCookie *securecookie.SecureCookie
}

// Middleware will perform the validation of reCAPTCHA challenge that the user should solve before passing
// control to the actual function that processes the full request and returns the result.
// TODO Don't hardcode the IPAddress when verifying the challenge
func (middle RecaptchaMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middle.ValidateRecaptchaCookie(r) == false {
			challenge := r.URL.Query().Get("c")

			catpcha := recaptcha.Recaptcha{PrivateKey: middle.Configuration.Recaptcha.PrivateKey}
			recaptchaResponse, _ := catpcha.Verify(challenge, r.RemoteAddr)

			if recaptchaResponse.Success == false {
				w.WriteHeader(403)
				return
			}
		}

		// FIXME We are generating a new cookie per request and thus avoiding a context check in the next function.
		ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
		secureRecaptchaCookie, _ := middle.GenerateCookie(ipAddress + r.UserAgent())
		ctx := context.WithValue(r.Context(), "recaptcha", secureRecaptchaCookie)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// DisplayRecaptcha verifies if the request contains a valid application reCAPTCHA cookie
// TODO Make this work with ReadCookie in the middleware
func DisplayRecaptcha(req *http.Request) bool {
	_, err := req.Cookie("reCAPTCHA")

	if err != nil {
		return true
	}

	return false
}

// GenerateCookie encodes the value passed as a parameter and returns a cookie with that encoded string as a value
func (recaptcha *RecaptchaMiddleware) GenerateCookie(value string) (http.Cookie, error) {
	cookie := http.Cookie{}
	encoded, err := recaptcha.SecureCookie.Encode(recaptcha.Configuration.RecaptchaCookie.Name, value)

	if err != nil {
		return cookie, err
	}

	cookie.Name = recaptcha.Configuration.RecaptchaCookie.Name
	cookie.Value = encoded
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Expires = time.Now().Add(24 * time.Hour)

	return cookie, err
}

// ReadCookie reads and validates the cookie that was passed as value
func (m *RecaptchaMiddleware) ReadCookie(r *http.Request) (string, error) {
	decodedValue := "";
	cookie, err := r.Cookie(m.Configuration.RecaptchaCookie.Name)

	if err != nil {
		return decodedValue, err
	}

	err = m.SecureCookie.Decode(m.Configuration.RecaptchaCookie.Name, cookie.Value, &decodedValue)

	if err != nil {
		return decodedValue, err
	}

	return decodedValue, nil
}

// ValidateRecaptchaCookie validates that the cookies's decoded contents contain the correct information and is valid.
func (m *RecaptchaMiddleware) ValidateRecaptchaCookie(r * http.Request) bool {
	decoded, _ := m.ReadCookie(r)
	ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
	return decoded == (ipAddress + r.UserAgent())
}
