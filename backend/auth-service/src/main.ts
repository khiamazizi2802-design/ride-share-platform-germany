import express from 'express';
import * as dotenv from 'dotenv';

dotenv.config();

const app = express();
const port = process.env.PORT || 3000;

app.use(express.json());

app.get('/health', (req, res) => {
  res.json({ status: 'UP', service: 'auth-service' });
});

app.listen(port, () => {
  console.log('Auth service listening at http://localhost:' + port);
});
