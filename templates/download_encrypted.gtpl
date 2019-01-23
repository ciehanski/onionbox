<!DOCTYPE html>
<html lang="en">
    <head>
        <title>onionbox - Download Encrypted</title>
        <meta charset="UTF-8">
    </head>
    <body>
        <h2>Click below to download your files securely.</h2>
        <form action="/" method="post">
            <input type="hidden" name="token" value="{{.}}" required/>
            <h5>Password:</h5>
            <input type="password" name="password"><br>
            <input type="submit">
        </form>
    </body>
</html>