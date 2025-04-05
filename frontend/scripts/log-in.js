const logInButton = document.getElementById('logInButton');
logInButton.addEventListener('click', logIn);
function logIn() {
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
            throw Error(response.text());
        }
        return response.json();
    })
    .then((data) => {
        //Store jwt token somewhere idk
        document.defaultView.localStorage.setItem('token', data);
        document.defaultView.localStorage.setItem('username', document.getElementById('username').value);
        //fix page
        document.getElementById('logInHeader').innerText = 'Logged in successfully';
        document.getElementById('username').remove();
        document.getElementById('password').remove();
        document.getElementById('logInButton').remove();
        const chatLink = document.createElement('a');
        chatLink.href = 'chats.html';
        chatLink.innerText = 'Get started with chatting!';
        document.getElementsByClassName('form')[0].appendChild(chatLink);
    })
    .catch((error) => console.log(error.Error));  
}    