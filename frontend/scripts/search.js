if (document.defaultView.localStorage.getItem('token')) {
    const elements = document.getElementsByClassName('h-button')
    elements[0].remove();
    elements[0].remove();
    const logoutButton = document.createElement('a');
    logoutButton.innerText = 'Log out';
    logoutButton.className = 'h-button';
    logoutButton.href = 'main.html';
    logoutButton.addEventListener('click', logOut);
    document.getElementById('header').appendChild(logoutButton);
}
const buttons =  document.getElementsByClassName('h-button');

for (let i = 0; i < buttons.length; i++) {
    buttons[i].addEventListener('mouseenter', function () {
        buttons[i].style.backgroundColor = 'rgb(80, 87, 122)';
    });
    buttons[i].addEventListener('mouseleave', function () {
        buttons[i].style.backgroundColor = 'rgb(64, 66, 88)';
    });
}

//Temporary
const user = document.defaultView.localStorage.getItem('username');
console.log(user);

function logOut() {
    document.defaultView.localStorage.removeItem('token');
    document.defaultView.localStorage.removeItem('username');
    window.location.href = '/';   
}

const searchButton = document.getElementById('search-button');
const inputSearch = document.getElementById('search-input');
searchButton.addEventListener('click', search); 

function search() {
    fetch(`${host}/user/${inputSearch.value}`)
    .then(response => response.json())
    .then(data => {
        const div = document.createElement('div');
        const e = document.createElement('p');
        const b = document.createElement('button');
        b.innerText = 'Text';
        b.addEventListener('click', function() {
            window.location.href = `/chats.html`;
        });
        e.innerText = data.username;
        div.appendChild(e);
        div.appendChild(b);
        document.getElementById('search').appendChild(div);
    })
    .catch(error => {
        const div = document.createElement('div');
        const e = document.createElement('p');
        e.innerText = `User not found :(`;
        div.appendChild(e);
        document.getElementById('search').appendChild(div);
    });
    inputSearch.value = '';
}
