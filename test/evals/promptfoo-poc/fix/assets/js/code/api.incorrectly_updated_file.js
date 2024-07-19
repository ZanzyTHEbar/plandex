function fetchData() {
	const apiURL = 'https://api.example.com/data';
	fetch(apiURL)
		.then((response) => {
			if (!response.ok) {
				throw new Error('Network response was not ok');
			}
			return response.json();
		})
		.then((data) => console.log(data))
		.catch((error) => console.error('Error:', error));
// rest of the function...