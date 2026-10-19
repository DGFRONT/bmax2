// Данные приложения
const AppData = {
    // Глава региона (всегда первый)
    governor: {
        id: 'governor',
        name: 'Игорь Бабушкин',
        role: 'Губернатор',
        avatar: 'assets/images/governor.jpg', // ЗАМЕНИТЕ на путь к вашему фото
        subscribers: 50000,
        deepLink: 'max://user/sergeimeniaylo',
        category: 'official'
    },

    // Популярные каналы (топ-4 по подписчикам)
    popularChannels: [
        {
            id: 'min_obr',
            name: 'Министерство образования',
            icon: '🏛️',
            subscribers: 35000,
            deepLink: 'max://user/sergeimeniaylo',
            category: 'ministries'
        },
        {
            id: 'min_health',
            name: 'Министерство здравоохранения',
            icon: '🏥',
            subscribers: 32000,
            deepLink: 'https://max.ru/channel_bot?startapp=channel_min_health',
            category: 'health'
        },
        {
            id: 'min_social',
            name: 'Министерство соцзащиты',
            icon: '👥',
            subscribers: 28000,
            deepLink: 'https://max.ru/channel_bot?startapp=channel_min_social',
            category: 'ministries'
        },
        {
            id: 'mfc',
            name: 'МФЦ',
            icon: '👨‍💼',
            subscribers: 25000,
            deepLink: 'https://max.ru/channel_bot?startapp=channel_mfc',
            category: 'mfc'
        }
    ],

    // Категории блоков
    categories: [
        {
            id: 'health',
            title: 'Здравоохранение',
            icon: '',
            large: false
        },
        {
            id: 'education',
            title: 'Образование',
            icon: '📚',
            large: false
        },
        {
            id: 'housing',
            title: 'ЖКХ',
            icon: '🏠',
            large: false
        },
        {
            id: 'mfc',
            title: 'МФЦ',
            icon: '👨‍💼',
            large: false
        },
        {
            id: 'media',
            title: 'СМИ',
            icon: '📰',
            large: false
        },
        {
            id: 'safety',
            title: 'Безопасность',
            icon: '🛡️',
            large: false
        },
        {
            id: 'fuel',
            title: 'Топливо',
            icon: '⛽',
            large: true
        },
        {
            id: 'other',
            title: 'Остальные ресурсы',
            icon: '📋',
            large: true
        }
    ],

    // Организации по категориям (ЗАПОЛНИТЕ своими данными)
    organizations: {
        health: [
            {
                id: 'hospital_1',
                name: 'Городская больница №1',
                icon: '',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_hospital_1',
                category: 'Здравоохранение'
            },
            {
                id: 'polyclinic',
                name: 'Поликлиника',
                icon: '‍⚕️',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_polyclinic',
                category: 'Здравоохранение'
            }
            // Добавьте остальные организации здравоохранения
        ],
        education: [
            {
                id: 'school_1',
                name: 'Школа №1',
                icon: '',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_school_1',
                category: 'Образование'
            },
            {
                id: 'kindergarten',
                name: 'Детский сад',
                icon: '🧸',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_kindergarten',
                category: 'Образование'
            }
            // Добавьте остальные организации образования
        ],
        housing: [
            {
                id: 'vodokanal',
                name: 'Водоканал',
                icon: '💧',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_vodokanal',
                category: 'ЖКХ'
            },
            {
                id: 'teploseti',
                name: 'Тепловые сети',
                icon: '',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_teploseti',
                category: 'ЖКХ'
            }
            // Добавьте остальные организации ЖКХ
        ],
        mfc: [
            {
                id: 'mfc_main',
                name: 'МФЦ Центральный',
                icon: '🏢',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_mfc_main',
                category: 'МФЦ'
            }
            // Добавьте остальные МФЦ
        ],
        media: [
            {
                id: 'tv_channel',
                name: 'Региональное ТВ',
                icon: '📺',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_tv',
                category: 'СМИ'
            },
            {
                id: 'radio',
                name: 'Региональное радио',
                icon: '📻',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_radio',
                category: 'СМИ'
            }
            // Добавьте остальные СМИ
        ],
        safety: [
            {
                id: 'mvd',
                name: 'МВД',
                icon: '🚔',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_mvd',
                category: 'Безопасность'
            },
            {
                id: 'mchs',
                name: 'МЧС',
                icon: '🚒',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_mchs',
                category: 'Безопасность'
            }
            // Добавьте остальные организации безопасности
        ],
        fuel: [
            {
                id: 'gas_station_1',
                name: 'АЗС №1',
                icon: '⛽',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_gas_1',
                category: 'Топливо'
            }
            // Добавьте остальные АЗС
        ],
        other: [
            {
                id: 'library',
                name: 'Библиотека',
                icon: '📚',
                deepLink: 'https://max.ru/channel_bot?startapp=channel_library',
                category: 'Остальное'
            }
            // Добавьте остальные организации
        ]
    },

    // Получить все организации плоским списком
    getAllOrganizations() {
        const all = [];
        for (const category in this.organizations) {
            all.push(...this.organizations[category]);
        }
        // Добавляем главу и популярные
        all.push(this.governor);
        all.push(...this.popularChannels);
        // Сортируем по алфавиту
        return all.sort((a, b) => a.name.localeCompare(b.name));
    },

    // Поиск организаций
    search(query) {
        const lowerQuery = query.toLowerCase();
        const all = this.getAllOrganizations();
        
        return all.filter(org => 
            org.name.toLowerCase().includes(lowerQuery) ||
            (org.category && org.category.toLowerCase().includes(lowerQuery))
        );
    },

    // Получить организации категории
    getCategoryOrganizations(categoryId) {
        return this.organizations[categoryId] || [];
    }
};