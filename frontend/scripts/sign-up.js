const signUpButton = document.getElementById('signUpButton')

signUpButton.addEventListener('click', signUp);

function signUp() {
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
           throw Error(response.text());
        }
        document.getElementById('signUpHeader').innerText = 'Signed up successfully';
        document.getElementById('username').remove();
        document.getElementById('password').remove();
        document.getElementById('signUpButton').remove();
        const loginLink = document.createElement('a');
        loginLink.href = 'login.html';
        loginLink.innerText = 'Login';
        document.getElementsByClassName('form')[0].appendChild(loginLink);
        return 'ok';
    })
    .then((data) => console.log(data))
    .catch((error) => console.log(error));       
}