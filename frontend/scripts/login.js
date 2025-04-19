document.getElementById('login-form').addEventListener('submit', function (e) {
    e.preventDefault()
    fetch(`${host}/login`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            username: document.getElementById('username').value,
            password: document.getElementById('password').value,
        }),
    })
        .then((response) => {
            if (!response.ok) {
                window.location.replace("login_fail.html")
            }
            return response.json();
        })
        .then((data) => {
            //Store jwt token somewhere idk
            document.defaultView.localStorage.setItem('token', data);
            document.defaultView.localStorage.setItem('username', document.getElementById('username').value);
            window.location.replace("chats.html")
        })
        .catch((error) => console.log(error.Error));
});
