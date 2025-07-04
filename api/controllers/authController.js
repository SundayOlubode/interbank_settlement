import express from 'express';
import { userManager } from '../data/userManager.js';


export class AuthController {
  constructor() {
    this.userManager = userManager;
    this.router = express.Router();
    this.initializeRoutes();
  }

  initializeRoutes() {
    this.router.post('/login', this.login.bind(this));
  }

  async login(req, res) {
    const { username, password } = req.body;
    const user = await this.userManager.getLoginUser(username, password);
    if (!user) {
      return res.status(401).json({ error: 'Invalid credentials' });
    }
    return res.status(200).json({
      message: 'Login successful',
      user: {
        id: user.id,
        firstname: user.firstname,
        lastname: user.lastname,
        accountNumber: user.accountNumber,
        balance: user.balance,        
        bankMSP: user.bankMSP,
      },
    }); 
  }
}