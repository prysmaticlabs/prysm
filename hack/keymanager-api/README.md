# Keymanager API

## Postman

You can use Postman to test the API. https://www.postman.com/

### Postman collection

In this package you will find the postman collection for the API for the keymanager-api. you can import this collection
in your own postman instance to test the API.

#### Updating the collection

the collection will need to be exported and overwritten to update the collection. A PR should be created once the file
is updated.

#### Authentication

Our keymanager API requires a valid bearer token to run the key manager. you can retrieve this bearer token from the url
displayed when running `validator --web`
i.e. `http://127.0.0.1:7500/initialize?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.BEuFWr-FpKIlJEIjgmujTQJlJF2aJRaUfFiuTBYVL3k`
The token can be copied and pasted into the authorization tab of each Postman request to authenticate.

