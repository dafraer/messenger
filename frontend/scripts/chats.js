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

if (document.defaultView.localStorage.getItem('chat')) {
       
}