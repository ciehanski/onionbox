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
        <form method="post">
            <input type="hidden" name="token" value="{{.}}" required/>
            <h4>Enter Password:</h4>
            <input type="password" name="password" required><br>
            <input type="submit" class="button" value="Download">
        </form>
		</center>
    </body>
</html>
<style type="text/css">
*{
 font-family: "Courier New", Courier, monospace;
}
</style>`
