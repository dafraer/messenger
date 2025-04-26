document.getElementById('signup-form').addEventListener('submit', function (e) {
    e.preventDefault();
    const pw = document.getElementById('password');
    const err = document.getElementById('password-error');
    if (pw.value.length < 8) {
        e.preventDefault();
        err.style.display = 'block';
        pw.focus();
    } else {
        err.style.display = 'none';
        fetch(`${host}/register`, {
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
                    window.location.replace("signup_fail.html")
                    throw Error(response.text());
                }
                window.location.replace("signup_success.html")
                return 'ok';
            })
            .then((data) => console.log(data))
            .catch((error) => console.log(error));
    }
});