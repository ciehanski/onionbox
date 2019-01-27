package templates

// Too avoid needing HTML files with the static binary
const UploadHTML = `<!DOCTYPE html>
<html lang="en">
    <head>
        <title>onionbox - Upload</title>
        <meta charset="UTF-8">
    </head>
    <body>
		<center>
        <h2>Please select the file you would like to securely share.</h2>
        <form method="post" enctype="multipart/form-data" action="/">
            <input type="file" name="files" required multiple><br>
            <input type="hidden" name="token" value="{{.}}" required/>
            <h4>Advanced Options</h4>
            <input type="checkbox" name="password_enabled">Protect with password?<br>
            <input type="password" name="password"><br>
            <input type="checkbox" name="limit_downloads">Limit downloads?<br>
            <input type="number" name="download_limit"><br>
            <input type="checkbox" name="expire">Automatically expire download link? (in minutes)<br>
            <input type="number" name="expiration_time"><br><br>
            <input type="submit" class="button" value="Upload">
        </form>
		</center>
    </body>
</html>`
