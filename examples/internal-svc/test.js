import { sleep } from 'k6';
import http from 'k6/http';

export const options = {
    vus: 10,
    duration: '2m',
};

export default function () {
    http.get('http://app.amazing-app:80/');
    sleep(5);
}
