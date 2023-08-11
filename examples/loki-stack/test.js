import http from 'k6/http';
import { sleep } from 'k6';

export default function () {
    http.get('https://test-api.k6.io/public/crocodiles/');
    console.log("Made a request to test-api!")
    sleep(1);
}