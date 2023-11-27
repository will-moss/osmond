package main

// # -- Dependencies
import (
    "github.com/go-chi/chi/v5"              // Server Framework (base)
    "github.com/go-chi/chi/v5/middleware"   // Server Framework (utils)
    "net/http"                              // Status codes
    "net/url"                               // URL input validation

    "log"                                   // Logging
    "fmt"                                   // String interpolation
    "io"                                    // Saving file to disk
    "bufio"                                 // Reading file on disk
    "regexp"                                // Email input validation
    "strings"                               // Email split and pass as arguments + String ops
    "strconv"                               // Convert -string to -int
    "encoding/json"                         // Reply using JSON
    "net"                                   // Check port available
    "time"                                  // Timeout
    "errors"                                // SSL files check

    "github.com/wneessen/go-mail"           // SMTP automation
    "context"                               // SMTP automation

    "os"                                    // Environment variables + File ops
    "github.com/joho/godotenv"              // Environment loading from .env

    "github.com/go-shiori/go-readability"   // HTML parsing into clean articles
    "github.com/go-resty/resty/v2"          // Gotenberg automation
)

// # -- Useful methods

/*
  Given a KEY and a FALLBACK, returns the value stored in the
  environment variable associated with KEY, and FALLBACK otherwise
  Bonus : Removes surrounding quotes if any, for compatibility
  Bonus : Normalizes boolean synonyms into boolean strings
*/
func getEnv(key string, fallback ...string) string {
  value, exists := os.LookupEnv(key)
  if !exists {
    if len(fallback) > 0 {
      value = fallback[0]
    } else {
      value = ""
    }
  } else {
    // Quotes removal
    value = strings.Trim(value, "\"")

    // Boolean normalization
    mapping := map[string]string {
      "0":     "FALSE",
      "off":   "FALSE",
      "false": "FALSE",
      "1":     "TRUE",
      "on":    "TRUE",
      "true":  "TRUE",
    }
    normalized, isBool := mapping[strings.ToLower(value)]
    if isBool {
      value = normalized
    }
  }

  return value
}

/*
  Given a STR, returns whether the STR is a valid URL
  following the example : https://my-website.tld/x/y/z
*/
func isUrl(str string) bool {
  u, err := url.Parse(str)
  return err == nil && u.Scheme == "https" && u.Host != ""
}

/*
  Given a STR, returns whether the STR is a valid EMAIL
  following the example : my-email@my-domain.tld
*/
func isEmail(str string) bool {
  emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
  return emailRegex.MatchString(str)
}

/*
  Wrapper for strconv.Atoi, to swallow the error
*/
func atoi(str string) int {
  value, err := strconv.Atoi(str)
  if err != nil {
    value = 0
  }
  return value
}

/*
  Wrapper for json.Marshall, to ignore the error
*/
func toJSON(v any) []byte {
  j,_ := json.Marshal(v)
  return j
}

/*
  Handy method to reply JSON in chi
*/
func replyJSON(w http.ResponseWriter, code int, v any) {
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(code)
  w.Write(toJSON(v))
}

/*
  Handy method to reply Raw Text in chi
*/
func replyText(w http.ResponseWriter, code int, v string) {
  w.Header().Set("Content-Type", "text/plain")
  w.WriteHeader(code)
  w.Write([]byte(v))
}

/*
  Given a URL, downloads the response provided by the remote
  web server, and stores it to a temporary file, to finally
  return the absolute path to that file on the disk
*/
func downloadFile(url string) (string, error) {
  // Create a temporary file
  out, err := os.CreateTemp("", "*.osmond.html")
  if err != nil {
    return "", err
  }
  defer out.Close()

  // Set up the HTTP client
  client := &http.Client{}
  req, _ := http.NewRequest("GET", url, nil)

  // Support for custom user agent
  if getEnv("DOWNLOAD_USER_AGENT") != "" {
    req.Header.Add("User-Agent", getEnv("DOWNLOAD_USER_AGENT"))
  }

  // Perform the HTTP request
  resp, err := client.Do(req)
  if err != nil {
    return "", err
  }
  defer resp.Body.Close()

  // Ensure remote server's response is adequate
  if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("bad status: %s", resp.Status)
  }

  // Writer the body to file
  _, err = io.Copy(out, resp.Body)
  if err != nil  {
    return "", err
  }

  return out.Name(), nil
}

/*
  Given a FILEPATH to an HTML page, parses it, and returns
  a clean article content, in HTML format still
*/
func parsePage(filepath string) (*readability.Article, error) {
  // Locate and open the file on disk
  file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

  // Generate a dummy URL because go-readability doesn't support not using one
  dummy, _ := url.Parse("https://dummy.url/dummy/path")

  // Turn the opened File into a Reader
	reader := bufio.NewReader(file)

  // Parse file content
  article, _ := readability.FromReader(reader, dummy)

  return &article, nil
}

/*
  Given a stripped down Article, returns an HTML body
  of it, with its title, featured image, and content
*/
func articleToHTML(a *readability.Article) (string) {
  html :=
          `<!DOCTYPE html>
          <html>
            <head>
              <meta charset="UTF-8" />
            </head>
            <body>
              <article>
                <h1>%s</h1>%s
                <div>%s</div>
              </article>
            </body>
          </html>`

  var imageFragment string
  if a.Image != "" {
    imageFragment = fmt.Sprintf(`<img src="%s" alt="Article's featued image" />`, a.Image)
  }

  return fmt.Sprintf(html, a.Title, imageFragment, a.Content)
}

/*
  Given a STR, stores it to a temp file on the disk
  and returns the absolute path to it
*/
func storeTemporarily(str string) (string, error) {
  // Create a temporary file
  out, err := os.CreateTemp("", "*.osmond.pdf")
  if err != nil {
    return "", err
  }
  defer out.Close()

  _, err = out.WriteString(str)
  if err != nil  {
    return "", err
  }

  return out.Name(), nil
}

/*
  Given a URL, sends that URL to a Gotenberg instance via HTTP Request
  and retrieves the generated PDF data, then stores it to a temp file,
  and eventually returns the path to that file on disk
*/
func convertToPdf(resource string) (string, error) {
  // Create a temporary file
  out, err := os.CreateTemp("", "*.osmond.pdf")
  if err != nil {
    return "", err
  }
  defer out.Close()

  // Set up a new HTTP client
  client := resty.New()

  // Set up and perform the request
  var response *resty.Response
  if isUrl(resource) { // Case when Gotenberg will download the remote page using the URL
    formData := map[string]string{
      "url": resource,
    }

    response, err = client.R().
      SetOutput(out.Name()).
      SetMultipartFormData(formData).
      Post(fmt.Sprintf("http://%s:%s/forms/chromium/convert/url", getEnv("GOTENBERG_HOST"), getEnv("GOTENBERG_PORT")))

  } else { // Case when Gotenberg will convert a given HTML file (generated by us previously by stripping down the page)
    file, errf := os.Open(resource)
    if errf != nil {
      return "", errf
    }
    defer file.Close()

    response, err = client.R().
      SetOutput(out.Name()).
      SetMultipartField("files", "index.html", "text/html", bufio.NewReader(file)).
      Post(fmt.Sprintf("http://%s:%s/forms/chromium/convert/html", getEnv("GOTENBERG_HOST"), getEnv("GOTENBERG_PORT")))
  }

  if err != nil {
    return "", err
  }

  if response.StatusCode() != http.StatusOK {
    return "", err
  }

  return out.Name(), nil
}

/*
  Given a STR (the future email recipient of the articles),
  returns a raw bookmarklet string to call the server from
  the browser
*/
func generateBookmarklet(settings map[string]string, secret string, recipient string) (string) {
  bookmarklet := `
    javascript:l=document.location.href,s="%s",r=encodeURIComponent("%s"),xhr=new XMLHttpRequest,xhr.open("POST","%s:%s/relay"),xhr.setRequestHeader("Content-Type","application/x-www-form-urlencoded"),xhr.send("secret="+s+"&recipient="+r+"&link="+l);
  `

  // Unminified :
  /*
    javascript:(() => {
        const l = document.location.href;
        const s = "%s";
        const r = encodeURIComponent("%s");

        xhr = new XMLHttpRequest();
        xhr.open("POST", "%s:%s/relay");
        xhr.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");

        xhr.send("secret=" + s + "&recipient=" + r + "&link=" + l);
    })();
  */

  bookmarkletCleaner := strings.NewReplacer(
    "\n", "",
    "\r", "",
  )
  bookmarklet = strings.TrimSpace(bookmarkletCleaner.Replace(bookmarklet))

  remote_port := settings["PORT"]
  if settings["PROXIED"] == "TRUE" {
    remote_port = settings["PROXY_PORT"]
  }

  return fmt.Sprintf(bookmarklet, secret, recipient, settings["HOST"], remote_port)
}

/*
  Perform checks to ensure the server is ready to start
  Returns an error if any condition isn't met
*/
func performVerifications(settings map[string]string) (bool, error) {
  // 1. Ensure creating a temp file works
  out, err := os.CreateTemp("", "*.osmond.check")
  if err != nil {
    return false, fmt.Errorf("Failed Verification : Temp file creation -> %s", err)
  }
  defer out.Close()

  // 2. Ensure we're able to download a remote file
  _, err = downloadFile("https://ip.me")
  if err != nil {
    return false, fmt.Errorf("Failed Verification : Remote file download -> %s", err)
  }

  // 3. Ensure SMTP credentials are good
  c, _ := mail.NewClient(
    getEnv("SMTP_HOST"),
    mail.WithPort(atoi(getEnv("SMTP_PORT"))),
    mail.WithUsername(getEnv("SMTP_USERNAME")),
    mail.WithPassword(getEnv("SMTP_PASSWORD")),
    mail.WithSMTPAuth(mail.SMTPAuthLogin),
  )
  if err := c.DialWithContext(context.TODO()); err != nil {
    return false, fmt.Errorf("Failed Verification : SMTP Connection -> %s", err)
  }

  // 4. Ensure server port is available
  l, err := net.Listen("tcp", fmt.Sprintf(":%s", settings["PORT"]))
  if err != nil {
    return false, fmt.Errorf("Failed Verification : Port binding -> %s", err)
  }
  defer l.Close()

  // 5. Ensure Gotenberg host is reachable when applicable
  if getEnv("DOWNLOAD_CONVERT_PDF") == "TRUE" {
    h, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", getEnv("GOTENBERG_HOST"), getEnv("GOTENBERG_PORT")), 1 * time.Second)
    if err != nil {
      return false, fmt.Errorf("Failed Verification : Gotenberg connection -> %s", err)
    }
    defer h.Close()
  }

	// 6. Ensure certificate and private key are provided
	if getEnv("SSL_ENABLED") == "TRUE" {
		if _, err := os.Stat("./certificate.pem"); errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("Failed Verification : Certificate file missing -> Please put your certificate.pem file next to the executable")
		}
		if _, err := os.Stat("./key.pem"); errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("Failed Verification : Private key file missing -> Please put your key.pem file next to the executable")
		}
	}

  return true, nil
}

// # -- Entry point
func main() {
  // Load settings via .env file
  err := godotenv.Load()
  if err != nil {
    log.Printf("No .env file provided, will continue with system env")
  }

  // Retrieve server settings & merge with defaults
  settings_server := map[string]string {
    "HOST":       getEnv("SERVER_HOST", "http://localhost"),
    "PORT":       getEnv("SERVER_PORT", "80"),
    "SECRET":     getEnv("SERVER_SECRET", "one-very-long-and-mysterious-secret"),
    "PROXIED":    getEnv("SERVER_PROXIED"),
    "PROXY_PORT": getEnv("SERVER_PROXY_PORT"),
  }

  if getEnv("SKIP_VERIFICATIONS") != "TRUE" {
    // Ensure everything is ready for our app
    log.Printf("Performing verifications before starting")
    _, err = performVerifications(settings_server)
    if err != nil {
      log.Printf("Error performing initial verifications, abort\n")
      log.Printf("%s", err)
      return
    }
  }


  // Instantiate server
  app := chi.NewRouter()

  // Set up basic middleware
  // app.Use(middleware.RequestID)
  if settings_server["PROXIED"] == "TRUE" {
    app.Use(middleware.RealIP)
  }
  app.Use(middleware.Logger)
  app.Use(middleware.Heartbeat("/heartbeat"))
  app.Use(middleware.Recoverer)

  // Define server routes & handlers
  app.Options("/relay", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
  })

  // POST /relay (expects body : secret=<S>&link=<L>&recipient=<R>)
  // Performs the link download, data extraction, optionally PDF conversion, and email send
	app.Post("/relay", func(w http.ResponseWriter, r *http.Request) {
    // Ensure secret is provided and matches the one in store
    secret := r.FormValue("secret")
    if secret != settings_server["SECRET"] {
      w.WriteHeader(http.StatusForbidden)
      return
    }

    // Ensure link is provided and has proper format
    link := r.FormValue("link")
    if !isUrl(link) {
      w.WriteHeader(http.StatusBadRequest)
      return
    }

    // Ensure recipient is provided and has proper format
    recipient := r.FormValue("recipient")
    if !isEmail(recipient) {
      w.WriteHeader(http.StatusBadRequest)
      return
    }

    // Download the page provided under link
    filepath, err := downloadFile(link)
    if err != nil {
      log.Printf("Error downloading page, abort")
      log.Printf("%s", err)
      log.Printf("Happened while working on : %s", link)
      replyJSON(w, http.StatusInternalServerError, map[string]any{ "error": err })
      return
    }

    // Retrieve the article's content stripped down
    article, err := parsePage(filepath)
    if err != nil {
      log.Printf("Error while parsing page with go-readability, abort")
      log.Printf("%s", err)
      log.Printf("Happened while working on : %s", link)
      replyJSON(w, http.StatusInternalServerError, map[string]any{ "error": err })
      return
    }

    // Convert the article to PDF if enabled
    attachment := ""
    if getEnv("DOWNLOAD_CONVERT_PDF") == "TRUE" {
      resourceToConvert := link

      if getEnv("DOWNLOAD_FORCE_READER") == "TRUE" {
        resourceToConvert, err = storeTemporarily(articleToHTML(article))
        if err != nil {
          log.Printf("Error while storing stripped down version on disk, abort")
          log.Printf("%s", err)
          log.Printf("Happened while working on : %s", link)
          replyJSON(w, http.StatusInternalServerError, map[string]any{ "error": err })
          return
        }
      }

      attachment, err = convertToPdf(resourceToConvert)

      if err != nil {
        log.Printf("Error while converting page via Gotenberg, abort ")
        log.Printf("%s", err)
        log.Printf("Happened while working on : %s", link)
        replyJSON(w, http.StatusInternalServerError, map[string]any{ "error": err })
        return
      }
    }

    // Start to set up the email
    m := mail.NewMsg()

    // Set up email subject
    switch true {
      // Case when auto-generated from article
      case getEnv("EMAIL_AUTO_SUBJECT", "TRUE") == "TRUE":
        m.Subject(fmt.Sprintf("%s - %s", article.SiteName, article.Title))

      // Case when provided by the settings
      case getEnv("EMAIL_FORCE_SUBJECT") != "":
        subjectTemplate := getEnv("EMAIL_FORCE_SUBJECT")
        // Support for templated subject
        if strings.HasPrefix(subjectTemplate, "T:") {
          subjectInterpolator := strings.NewReplacer(
              "<TITLE>", article.Title,
              "<AUTHOR>", article.Byline,
              "<SITE>", article.SiteName,
          )
          m.Subject(subjectInterpolator.Replace(subjectTemplate[2:]))
        } else {
          m.Subject(subjectTemplate)
        }

      // Case when neither provided nor auto-generated
      default:
        m.Subject("Read later - Your new article")
    }

    // Set up PDF attachment
    if attachment != "" {
      m.AttachFile(attachment)
    }

    // Set up the last email settings
    if getEnv("SMTP_FROM") != "" {
      m.From(getEnv("SMTP_FROM"))
    } else {
      m.From(getEnv("SMTP_USERNAME"))
    }

    m.To(strings.Split(recipient, ",")...)
    m.SetBodyString("text/html", articleToHTML(article))

    // Send the email
    settings_smtp := map[string]string {
      "HOST": getEnv("SMTP_HOST"),
      "PORT": getEnv("SMTP_PORT"),
      "USER": getEnv("SMTP_USERNAME"),
      "PASS": getEnv("SMTP_PASSWORD"),
    }

    c, _ := mail.NewClient(
      settings_smtp["HOST"],
      mail.WithPort(atoi(settings_smtp["PORT"])),
		  mail.WithUsername(settings_smtp["USER"]),
      mail.WithPassword(settings_smtp["PASS"]),
      mail.WithSMTPAuth(mail.SMTPAuthLogin),
    )
    if err := c.DialAndSend(m); err != nil {
      log.Printf("Error while sending email, abort")
      log.Printf("%s", err)
      log.Printf("Happened while working on : %s", link)
      replyJSON(w, http.StatusInternalServerError, map[string]any{ "error": err })
      return
    }

    w.WriteHeader(http.StatusOK)
	})

  // POST /bookmarklet (expects body : secret=<S>&recipient=<R>)
  // Returns the bookmarket with SECRET and RECIPIENT fields filled in
  app.Post("/bookmarklet", func(w http.ResponseWriter, r *http.Request) {
    // Ensure secret is provided and matches the one in store
    secret := r.FormValue("secret")
    if secret != settings_server["SECRET"] {
      w.WriteHeader(http.StatusForbidden)
      return
    }

    // Ensure recipient is provided and has proper format
    recipient := r.FormValue("recipient")
    if !isEmail(recipient) {
      w.WriteHeader(http.StatusBadRequest)
      return
    }

    replyText(w, http.StatusOK, generateBookmarklet(settings_server, secret, recipient))
  })

  // GET /bookmarket
  // Returns the default template with placeholder fields
  app.Get("/bookmarklet", func(w http.ResponseWriter, r *http.Request) {
    replyText(w, http.StatusOK, generateBookmarklet(settings_server, "<SECRET>", "<RECIPIENT>"))
  })

  log.Printf("Server starting on port %s", settings_server["PORT"])

  if getEnv("SHOW_BOOKMARKLET", "TRUE") == "TRUE" {
    log.Printf("Here's your bookmarklet :\n===\n%s\n===", generateBookmarklet(settings_server, settings_server["SECRET"], "<YOUR-EMAIL>"))
    log.Printf("Tip 1 : Don't forget to change <YOUR-EMAIL> with your actual email address")
    log.Printf("Tip 2 : In many terminals, you can simply select the bookmarklet's code and drag it directly into your browser")
  }

  // Start server
  if getEnv("SSL_ENABLED") == "TRUE" {
    http.ListenAndServeTLS(fmt.Sprintf(":%s", settings_server["PORT"]), "certificate.pem", "key.pem", app)
  } else {
    http.ListenAndServe(fmt.Sprintf(":%s", settings_server["PORT"]), app)
  }
}
