require('dotenv').config();
const express = require('express');
const cors = require('cors');
const jwt = require('jsonwebtoken');
const fs = require('fs').promises;
const path = require('path');
const validateMaxInitData = require('./validate');

const app = express();
app.use(express.json());
app.use(cors());

const authenticate = (req, res, next) => {
    const token = req.headers.authorization?.split(' ')[1];
    if (!token) return res.status(401).json({ error: 'Нет токена' });

    try {
        const decoded = jwt.verify(token, process.env.JWT_SECRET);
        req.user = decoded;
        next();
    } catch (e) {
        res.status(403).json({ error: 'Невалидный токен' });
    }
};

app.post('/api/auth', async (req, res) => {
    const { initData } = req.body;
    if (!initData) return res.status(400).json({ error: 'Нет initData' });

    const isValid = validateMaxInitData(initData, process.env.BOT_TOKEN);
    if (!isValid) {
        console.warn(`[WARN] Невалидный initData от IP: ${req.ip}`);
        return res.status(403).json({ error: 'Невалидные данные MAX' });
    }

    const params = new URLSearchParams(initData);
    const userStr = params.get('user');
    const startParam = params.get('start_param') || 'none';
    let userData = {};
    try { userData = JSON.parse(userStr); } catch(e) {}

    console.log(`[INFO] Вход: ID=${userData.id}, Username=${userData.username}, StartParam=${startParam}, IP=${req.ip}`);

    const token = jwt.sign(
        { userId: userData.id, username: userData.username },
        process.env.JWT_SECRET,
        { expiresIn: '24h' }
    );

    res.json({ token, user: { id: userData.id, username: userData.username } });
});

app.get('/api/channels', authenticate, async (req, res) => {
    try {
        const dataPath = path.join(__dirname, 'channels.json');
        const rawData = await fs.readFile(dataPath, 'utf-8');
        const channelsData = JSON.parse(rawData);
        res.json(channelsData);
    } catch (error) {
        console.error('[ERROR] Ошибка чтения channels.json:', error);
        res.status(500).json({ error: 'Ошибка сервера' });
    }
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
    console.log(`✅ Backend запущен на порту ${PORT}`);
});