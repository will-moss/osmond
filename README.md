<p align="center">
<h1 align="center">Osmond</h1>
<p align="center">Self-hostable alternative to EmailThis</p>
</p>


## Introduction

Osmond is a small, simple, and self-hostable service that enables you to send ad-free pages and articles to your email inbox. It is an attempt at recreating the [EmailThis](https://www.emailthis.me/) service from scratch, while making it open and free for everyone to use.


## Features

Osmond has all these features implemented :
- Page article's extraction and sending as HTML by email
- PDF conversion and sending as attachment by email
- Automatic / Dynamically Personalized email subject
- Called via a (generated and displayed) bookmarklet
- Support for multiple email recipients
- Support for extensive configuration with `.env`
- Support for HTTP and HTTPS
- Support for standalone / proxy deployment

On top of these, one may appreciate the following characteristics :
- Written in Go
- Holds in a single file with few dependencies
- Holds in a ~14 MB compressed Docker image

For more information, read about [Configuration](#configuration) and [API Reference](#api-reference).


## Deployment and Examples

### Deploy with Docker

To help you get started quickly, multiple example `docker-compose` files are located in the ["examples/"](examples) directory.

Here's a description of every example :

- `docker-compose.simple.yml`: Run Osmond as a front-facing service on port 80., with environment variables supplied in the `docker-compose` file directly.

- `docker-compose.volume.yml`: Run Osmond as a front-facing service on port 80, with environment variables supplied as a `.env` file mounted as a volume.

- `docker-compose.gotenberg.yml`: Run Osmond as a front-facing service on port 80, with a companion Gotenberg container running to convert pages into PDF files before being sent by email.

- `docker-compose.ssl.yml`:  Run Osmond as a front-facing service on port 443, listening for HTTPS requests, with certificate and private key provided as mounted volumes.

- `docker-compose.proxy.yml`: A full setup with Osmond running on port 80, behind a proxy listening on port 443, and Gotenberg running as well while not being exposed to external requests.

When your `docker-compose` file is on point, you can use the following commands :
```sh
# Run Osmond in the current terminal (useful for debugging)
docker-compose up

# Run Osmond in a detached terminal (most common)
docker-compose up -d

# Show the logs written by Osmond (useful for debugging)
docker logs <NAME-OF-YOUR-CONTAINER>
```

### Deploy as a standalone application

Deploying Osmond as a standalone application assumes the following prerequisites :
- You have Go installed on your server
- You have properly filled your `.env` file
- Your DNS and networking configuration is on point

When all the prerequisites are met, you can run the following commands in your terminal :

```sh
# Retrieve the code
git clone https://github.com/will-moss/osmond
cd osmond

# Create a new .env file
cp sample.env .env

# Edit .env file ...

# Build the code into an executable
go build -o osmond main.go

# Option 1 : Run Osmond in the current terminal
./osmond

# Option 2 : Run Osmond as a background process
./osmond &

# Option 3 : Run Osmond using screen
screen -S osmond
./osmond
<CTRL+A> <D>
```

## Configuration

To run Osmond, you will need to set the following environment variables in a `.env` file located next to your executable :

> **Note :** Regular environment variables provided on the commandline work too

| Parameter               | Type      | Description                | Default |
| :---------------------- | :-------- | :------------------------- | ------- |
| `SSL_ENABLED`           | `boolean` | Whether HTTPS should be used in place of HTTP. When configured, Osmond will look for `certificate.pem` and `key.pem` next to the executable for configuring SSL. Note that if Osmond is behind a proxy that already handles SSL, this should be set to `false`. | False        |
| `SERVER_HOST`           | `string`  | The remote host used to reach Osmond from the outside, including the protocol, and without trailing slash. (e.g. http://your-server.tld) | http://localhost        |
| `SERVER_PORT`           | `integer` | The port Osmond listens on. | 80        |
| `SERVER_SECRET`         | `string`  | The secret used to secure your Osmond instance against bots / malicious usage. | one-very-long-and-mysterious-secret        |
| `SERVER_PROXIED`        | `boolean` | Whether Osmond sits behind a proxy. | False        |
| `SERVER_PROXY_PORT`     | `integer` | The port used by the front-facing proxy. | None         |
| `DOWNLOAD_USER_AGENT`   | `string`  | The user agent Osmond should use when downloading remote pages before sending them by email | Chrome's default user agent on Mac        |
| `DOWNLOAD_CONVERT_PDF`  | `boolean` | Whether Osmond should convert pages to PDF and send them by email afterwards. (Requires a Gotenberg instance) | False        |
| `DOWNLOAD_FORCE_READER` | `boolean` | By default, Gotenberg will turn the full web page into PDF. When set to `true`, this setting will make Gotenberg use the `Reader mode` version of the page to convert it, instead of using the whole page with ads and distraction. | False        |
| `SMTP_HOST`             | `string`  | Your SMTP host for sending emails | None        |
| `SMTP_PORT`             | `integer` | Your SMTP port | None        |
| `SMTP_USERNAME`         | `string`  | Your SMTP username | None        |
| `SMTP_PASSWORD`         | `string`  | Your SMTP password | None        |
| `SMTP_FROM`             | `string`  | The email address to send the emails from. | Uses `SMTP_FROM` value       |
| `GOTENBERG_HOST`        | `string`  | The host used to reach Gotenberg's instance, without protocol. | None        |
| `GOTENBERG_PORT`        | `integer` | The port used to reach Gotenberg's instance | None        |
| `EMAIL_AUTO_SUBJECT`    | `boolean` | Whether Osmond should automatically set the emails' subject using the pages' metadata. (Current format : %SITE_NAME% - %PAGE_TITLE% ) | True        |
| `EMAIL_FORCE_SUBJECT`   | `string`  | The subject to use in place of the automatically generated one. Will only work if `EMAIL_AUTO_SUBJECT` is set to `false`. Supports templating, example: "T:\<TITLE\> --- \<AUTHOR\> --- \<SITE\>" | None        |
| `SKIP_VERIFICATIONS`    | `boolean` | Whether Osmond should skip startup verification checks before running the HTTP(S) server. | False        |
| `SHOW_BOOKMARKLET`      | `boolean` | Whether Osmond should show the bookmarklet's snippet at startup | True

> **Note :** Boolean values are case-insensitive, and can be represented via "ON" / "OFF" / "TRUE" / "FALSE" / 0 / 1.

> **Tip :** You can use the `email+folder@domain.tld` notation to automatically put your pages in a dedicated folder of your email inbox.

> **Tip :** You can generate a random secret with the following command :

```sh
head -c 1024 /dev/urandom | base64 | tr -cd "[:lower:][:upper:][:digit:]" | head -c 32
```


## API Reference

Osmond exposes the following API :

#### Heartbeat

```
  GET /heartbeat
```
```
  .
```

#### Get the bookmarklet template

```
  GET /bookmarklet
```
```javascript
javascript:l=document.location.href,s="<SECRET>",r=encodeURIComponent("<RECIPIENT>"),xhr=new XMLHttpRequest,xhr.open("POST","<HOST>:<PORT>/relay"),xhr.setRequestHeader("Content-Type","application/x-www-form-urlencoded"),xhr.send("secret="+s+"&recipient="+r+"&link="+l);
```

#### Get the actual bookmarklet code for your instance

```
  POST /bookmarklet
```

| Parameter | Type     | Description                       |
| :-------- | :------- | :-------------------------------- |
| `secret`  | `string` | **Required.** Your server secret |
| `recipient` | `string`| **Required.** The email to send the pages / PDF to (can be a list of comma-separated emails). |


> **Note :** POST data should be sent in the body, using `x-www-form-urlencoded` format.

#### Send a page by email

```
  POST /relay
```

| Parameter | Type     | Description                       |
| :-------- | :------- | :-------------------------------- |
| `secret`  | `string` | **Required.** Your server secret |
| `recipient` | `string`| **Required.** The email to send the pages / PDF to (can be a list of comma-separated emails). |
| `link` | `string` | **Required.** The link of the page to download and send by email (should be an HTTPS url).

> **Note :** POST data should be sent in the body, using `x-www-form-urlencoded` format.

> **Note :** Most often, Osmond returns nothing more than a status code


## Troubleshoot

Should you encounter any issue running Osmond, please refer to the following common problems that may occur.

> If none of these matches your case, feel free to open an issue.


#### Osmond is unreachable over HTTP / HTTPS

Please make sure that the following requirements are met :

- If Osmond runs as a standalone application without proxy :
    - Make sure your server / firewall accepts incoming connections on Osmond's port.
    - Make sure your DNS configuration is correct. (Usually, such record should suffice : `A osmond XXX.XXX.XXX.XXX` for `https://osmond.your-server-tld`)
    - Make sure your `.env` file is well configured according to the [Configuration](#configuration) section.

- If osmond runs behind Docker / a proxy :
    - Perform the previous (standalone) verifications first.
    - Make sure that `SERVER_PORT` (Osmond's port) and `SERVER_PROXY_PORT` (the proxy's port) are well set in `.env`.
    - Check your proxy forwarding rules.

In any case, the crucial part is [Configuration](#configuration).


#### When I click on the bookmarklet, nothing happens

The current bookmarklet implementation just works, and won't display any confirmation message. The confirmation that everything works will be in your email inbox.

Now, if no email is received, you should check your browser's developer console, especially the `Network` and `Console` tabs. If an error occurs, it will show in the `Console`. You can start debugging / opening an issue from there.

Most often, the problem comes from a `Content Security Policy` set on the website you're browsing. Unfortunately, this effectively blocks the bookmarklet (although the specification for that feature states explicitly that bookmarklet shouldn't be blocked). The only ways to fix that is to either disable the `Content Security Policy` feature in your browser (not recommended), or creating a browser extension that bypasses it. (Let me know if that should be added to Osmond's roadmap!)


#### I lost my bookmarklet, how to get it back?

You've got multiple ways to get your bookmarklet back :
- Perform a `GET /bookmarklet` request on Osmond. It will display a bookmarklet template to fill in with you secret and email recipient.
- Perform a `POST /bookmarklet` request on Osmond, providing `secret` and `recipient` in the body. It will display a complete bookmarklet.
- Run `docker logs <YOUR-CONTAINER>` on your Osmond container. You should see your bookmarklet's template if you enabled `SHOW_BOOKMARKLET` environment variable.


#### Osmond fails to download my page

Many reasons could explain that, but most often it boils down to :
- Your page is protected (behind a login screen, or behind a captcha, or behind a VPN, or else).
- Your page is too big (you may need to adjust your server settings or your Gotenberg instance if you have PDF conversion enbled).
- Your page is loaded as a part of a Single Page Application that can't fully load without a full-functioning headless browser.

Please note that Osmond isn't (yet?) able to download protected pages, as that would require impersonating you.

#### How to send a page to my instance?

You're free to use the tool / app of your choice, and any HTTP client will do.
Now, most certainly, you'd want to go with a bookmarklet. You can create one in your browser by creating a regular bookmark, and copy-paste Osmond's bookmarklet in the `URL` parameter. Now, when clicking on the bookmark in your browser, Javascript will perform a POST request to your instance, hence triggering the page-to-email procedure.


#### Something else

Please feel free to open an issue, explaining what happens, and describing your environment.


## Credits

Hey hey ! It's always a good idea to say thank you and mention the people and projects that help us move forward.

Big thanks to the individuals / teams behind these projects :
- [Gotenberg](https://github.com/gotenberg/gotenberg) : For the HTML to PDF conversion.
- [go-readability](https://github.com/go-shiori/go-readability) : For the page's article extraction.
- The countless others!

And don't forget to mention Osmond if you like it or if it helps you in any way!
