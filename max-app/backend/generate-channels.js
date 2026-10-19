// generate-channels.js
const fs = require('fs');
const path = require('path');

// Импорт всех категорий из отдельных файлов
const government = require('./data/government');
const lawEnforcement = require('./data/law-enforcement');
const education = require('./data/education');
const health = require('./data/health');
const sport = require('./data/sport');
const culture = require('./data/culture'); 
const social = require('./data/social');
const selfgov = require('./data/selfgov');
const media = require('./data/media');
const bots = require('./data/bots');
const other = require('./data/other');

// Формирование структуры
const result = {
  governor: {
    id: "governor",
    name: "Глава РСО-Алания",
    username: "sergeimeniaylo",
    icon: "👤",
    avatar: "https://78.110.151.227.max.rsocit.ru/assets/images/gsm.jpg",
    subscribers: 50000,
    deepLink: "https://max.ru/sergeimeniaylo",
    category: "official"
  },
  popularChannels: [
    { 
      id: "gov", 
      name: "Правительство РСО-Алания", 
      username: "alania_gov", 
      icon: "🏛️", 
      avatar: "https://78.110.151.227.max.rsocit.ru/assets/images/building.jpg", 
      subscribers: 45000, 
      deepLink: "https://max.ru/alania_gov", 
      category: "government" 
    },
    { 
      id: "comdigital", 
      name: "Комитет цифрового развития РСО-Алания", 
      username: "id1513063914_gos", 
      icon: "💻", 
      avatar: "https://78.110.151.227.max.rsocit.ru/assets/images/prl.jpg", 
      subscribers: 35000, 
      deepLink: "https://max.ru/id1513063914_gos", 
      category: "government" 
    },
    { 
      id: "mchs", 
      name: "МЧС Северной Осетии", 
      username: "id1515900188_gos", 
      icon: "", 
      avatar: "https://78.110.151.227.max.rsocit.ru/assets/images/mchs.jpg", 
      subscribers: 38000, 
      deepLink: "https://max.ru/id1515900188_gos", 
      category: "emergency" 
    },
    { 
      id: "minzdrav", 
      name: "Минздрав РСО-Алания", 
      username: "minzdravrso", 
      icon: "", 
      avatar: "https://78.110.151.227.max.rsocit.ru/assets/images/building.jpg", 
      subscribers: 35000, 
      deepLink: "https://max.ru/minzdravrso", 
      category: "health" 
    },
    { 
      id: "mfc", 
      name: "МФЦ РСО-Алания", 
      username: "id1515922174_gos", 
      icon: "👨‍💼", 
      avatar: "https://78.110.151.227.max.rsocit.ru/assets/images/mfc.jpg", 
      subscribers: 28000, 
      deepLink: "https://max.ru/id1515922174_gos", 
      category: "mfc" 
    }
  ],
  categories: [
    { id: "government", title: "Органы власти РСО-Алания", icon: "🏛️", large: true },
    { id: "law", title: "Правоохранительные органы", icon: "️", large: false },
    { id: "parliament", title: "Парламент РСО-Алания", icon: "🏛️", large: false },
    { id: "emergency", title: "Экстренные службы", icon: "🚨", large: false },
    { id: "health", title: "Здравоохранение", icon: "🏥", large: false },
    { id: "mfc", title: "МФЦ", icon: "‍💼", large: false },
    { id: "education", title: "Образование и Наука", icon: "🎓", large: false },
    { id: "sport", title: "Спорт", icon: "⚽", large: false },
    { id: "culture", title: "Культура", icon: "", large: false },
    { id: "social", title: "Социальная защита", icon: "🤝", large: false },
    { id: "vodokanal", title: "ГУП Водоканал РСО-Алания", icon: "💧", large: true },
    { id: "fuel", title: "Топливные карты ЕИРЦ", icon: "", large: true },
    { id: "media", title: "СМИ", icon: "📰", large: true },
    { id: "selfgov", title: "Самоуправления", icon: "🏘️", large: true },
    { id: "bots", title: "Боты региона", icon: "🤖", large: true },
    { id: "other", title: "Прочее", icon: "📦", large: true }
  ],
  organizations: {
    government: [...government],
    law: [...lawEnforcement],
    parliament: [
      { id: "parlament", name: "Парламент Республики Северная Осетия-Алания", username: "id1502009231_gos", icon: "⚖️", avatar: "", deepLink: "https://max.ru/id1502009231_gos", category: "Парламент" },
      
    ],
    emergency: [
      { id: "mchs_channel", name: "МЧС по РСО", username: "mchs_rso_a", icon: "🚒", avatar: "", deepLink: "https://max.ru/mchs_rso_a", category: "Экстренные службы" },
      { id: "opershtab", name: "Оперативный штаб МинЖКХ", username: "opershtab_bot", icon: "⚠️", avatar: "", deepLink: "https://max.ru/opershtab_bot", category: "Экстренные службы" },
      { id: "signal_m", name: "СИГНАЛ-М", username: "scanmvd_bot", icon: "🚨", avatar: "", deepLink: "https://max.ru/scanmvd_bot", category: "Экстренные службы" },
      { id: "kiberpatrul", name: "КиберПатруль", username: "kiberpatrul_bot", icon: "🛡️", avatar: "", deepLink: "https://max.ru/kiberpatrul_bot", category: "Экстренные службы" },
      { id: "rkb", name: "РКБ", username: "rkb_notification_bot", icon: "🚑", avatar: "", deepLink: "https://max.ru/rkb_notification_bot", category: "Экстренные службы" },
      { id: "klinika_sogma", name: "Клиническая больница СОГМА", username: "klinika_sogma_bot", icon: "🏥", avatar: "", deepLink: "https://max.ru/klinika_sogma_bot", category: "Экстренные службы" },
      { id: "skmmc", name: "СК ММЦ (г. Беслан)", username: "skmmcmed_bot", icon: "🏥", avatar: "", deepLink: "https://max.ru/skmmcmed_bot", category: "Экстренные службы" }
    ],
    health: [...health],
    mfc: [
      { id: "mfc_channel", name: "МФЦ РСО-Алания (канал)", username: "id1515922174_gos", icon: "👨‍💼", avatar: "", deepLink: "https://max.ru/id1515922174_gos", category: "МФЦ" },
      { id: "family_mfc", name: "СЕМЕЙНЫЙ МФЦ РСО-АЛАНИЯ", username: "id1515917946_gos", icon: "👨👩‍👧", avatar: "", deepLink: "https://max.ru/id1515917946_gos", category: "МФЦ" }
    ],
    education: [...education],
    sport: [...sport],
    culture: [...culture], // <-- ТЕПЕРЬ БЕРЕТСЯ НАПРЯМУЮ ИЗ ФАЙЛА
    social: [...social],
    vodokanal: [
      { id: "vodokanal_bot", name: "ГУП «Водоканал» РСО-Алания", username: "gupvodokanal_bot", icon: "", avatar: "", deepLink: "https://max.ru/gupvodokanal_bot", category: "Водоканал" }
    ],
    fuel: [
      { id: "eirc_fuel", name: "Топливные карты ЕИРЦ", username: "eirc15", icon: "⛽", avatar: "", deepLink: "https://eirc15.ru/", category: "Топливные карты" }
    ],
    media: [...media],
    selfgov: [...selfgov],
    bots: [...bots],
    other: [...other]
  }
};

// Сортировка всех категорий по алфавиту
for (const cat in result.organizations) {
  result.organizations[cat].sort((a, b) => a.name.localeCompare(b.name, 'ru'));
}

// Удаление дубликатов по URL во всем объекте организаций
const seenUrls = new Set();
for (const cat in result.organizations) {
  result.organizations[cat] = result.organizations[cat].filter(org => {
    if (seenUrls.has(org.deepLink)) return false;
    seenUrls.add(org.deepLink);
    return true;
  });
}

// Запись в файл
fs.writeFileSync('/var/www/max-app/backend/channels.json', JSON.stringify(result, null, 2));

console.log('✅ channels.json успешно сгенерирован!');
console.log('Статистика по категориям:');
let total = 0;
for (const [cat, orgs] of Object.entries(result.organizations)) {
  console.log(`  ${cat.padEnd(15)}: ${orgs.length}`);
  total += orgs.length;
}
console.log(`\n ВСЕГО ОРГАНИЗАЦИЙ: ${total}`);