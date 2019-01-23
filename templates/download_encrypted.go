package templates

// Too avoid needing HTML files with the static binary
const DownloadHTML = `<!DOCTYPE html>
<html lang="en">
    <head>
        <title>onionbox - Download Encrypted</title>
        <meta charset="UTF-8">
    </head>
    <body>
        <center>
        <h2>Click below to download your files securely.</h2>
        <form action="/" method="post">
            <input type="hidden" name="token" value="{{.}}" required/>
            <h5>Password:</h5>
            <input type="password" name="password" required><br>
            <input type="submit" value="Download">
        </form>
		</center>
    </body>
</html>`
