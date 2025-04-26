//check login
if (document.defaultView.localStorage.getItem('token')) {
    const e = document.getElementById("username")
    e.innerText = document.defaultView.localStorage.getItem('username')
} else {
    const username = document.getElementById("username")
    const logoutButton = document.getElementById("logout_button")
    username.remove()
    logoutButton.className = "btn login"
    logoutButton.href = "login.html"
    logoutButton.innerText = "Login"
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


const searchButton = document.getElementById('search-button');
const inputSearch = document.getElementById('search-input');
searchButton.addEventListener('click', search); 

function search(e) {
    e.preventDefault()
    fetch(`${host}/user/${inputSearch.value}`)
    .then(response => response.json())
    .then(data => {
        document.defaultView.localStorage.setItem('chat', data.username);
        document.getElementsByClassName('user-list')[0].innerHTML = `
        <div class="user-item"><div class="user-info">
            <h4>${data.username}</h4>
            </div>
            <a href="chats.html" class="btn signup">Message</a>
        </div>`
    })
    .catch(error => {
        console.log(error)
        document.getElementsByClassName('user-list')[0].innerHTML = `
        <div class="user-item"><div class="user-info">
            <h4>User not found</h4>
            </div>
        </div>`
    });
    inputSearch.value = '';
}
