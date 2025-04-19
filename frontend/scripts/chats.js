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
if (document.defaultView.localStorage.getItem('chat')) {

}

