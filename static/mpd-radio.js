function openDialog(name) {
    document.getElementById(name).showModal()
}
function closeDialog(name) {
    let element = document.getElementById(name)
    console.debug(`closing ${element}`)
    element.close()
}