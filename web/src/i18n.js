import { ref } from 'vue'

export const LOCALES = [
  { id: 'uz', label: "O'zbek", flag: '🇺🇿' },
  { id: 'ru', label: 'Русский', flag: '🇷🇺' },
  { id: 'en', label: 'English', flag: '🇬🇧' },
]

export const locale = ref(localStorage.getItem('locale') || 'uz')
export function setLocale(l) { locale.value = l; localStorage.setItem('locale', l) }

const M = {
  uz: {
    'app.subtitle': 'Call markaz',
    'nav.dashboard': 'Boshqaruv paneli', 'nav.survey': 'Anketa', 'nav.calls': "Qo'ng'iroqlar", 'nav.surveyReport': 'Anketa hisoboti',
    'nav.servers': 'Serverlar', 'nav.staff': 'Hodimlar', 'nav.sessions': 'Sessiyalar', 'nav.analytics': 'Analitika', 'nav.feedback': 'Mijoz baholari', 'nav.auditLog': 'Amallar jurnali', 'nav.tv': 'TV ekran',
    'common.all': 'Hammasi', 'common.today': 'Bugun', 'common.yesterday': 'Kecha', 'common.week': 'Hafta',
    'common.month': 'Oy', 'common.custom': 'Maxsus', 'common.save': 'Saqlash', 'common.cancel': 'Bekor',
    'common.logout': 'Chiqish', 'common.theme.dark': 'Qora rejim', 'common.theme.light': 'Oq rejim',
    'role.admin': 'Administrator', 'role.operator': 'Operator',
    'dash.live': 'Jonli ulanish', 'dash.updating': 'Yangilanmoqda', 'dash.connecting': 'Ulanmoqda…',
    'dash.kpi.total': "Qo'ng'iroqlar", 'dash.kpi.in': 'Kiruvchi', 'dash.kpi.out': 'Chiquvchi',
    'dash.kpi.avg': "O'rtacha suhbat", 'dash.kpi.active': 'Faol operatorlar',
    'dash.hourly': 'Soatlik faollik', 'dash.operators': 'Operatorlar', 'dash.recent': "So'nggi qo'ng'iroqlar",
    'dash.incoming': 'Kiruvchi', 'dash.outgoing': 'Chiquvchi',
    'st.total': 'Jami', 'st.answered': 'Javob berildi', 'st.missed': "O'tkazib yuborilgan",
    'st.success': 'Muvaffaqiyatli', 'st.failed': "Dozvonilmadi", 'st.avgTalk': "O'rtacha suhbat",
    'st.totalTalk': 'Umumiy suhbat', 'st.surveys': 'Anketa to\'ldirilgan', 'st.opTable': 'Operatorlar statistikasi',
    'st.name': 'Ism', 'st.online': 'Onlayn', 'st.inCalls': 'Kiruvchi', 'st.outCalls': 'Chiquvchi', 'st.time': 'Vaqt',
    'tv.title': 'Operatorlar monitoringi', 'tv.onLine': 'Liniyada', 'tv.offLine': "Liniyada yo'q",
    'tv.talking': 'Gaplashyapti', 'tv.dnd': 'Pilot (DND)', 'tv.ringing': 'Jiringlamoqda', 'tv.status': 'Holat',
    'tv.unfilled': "To'ldirilmagan anketa", 'tv.servers': 'Serverlar', 'tv.missed': "O'tkazib yuborilgan", 'tv.unknown': "Noma'lum",
  },
  ru: {
    'app.subtitle': 'Колл-центр',
    'nav.dashboard': 'Панель', 'nav.survey': 'Анкета', 'nav.calls': 'Звонки', 'nav.surveyReport': 'Отчёт по анкетам',
    'nav.servers': 'Серверы', 'nav.staff': 'Сотрудники', 'nav.sessions': 'Сессии', 'nav.analytics': 'Аналитика', 'nav.feedback': 'Отзывы клиентов', 'nav.auditLog': 'Журнал действий', 'nav.tv': 'ТВ экран',
    'common.all': 'Все', 'common.today': 'Сегодня', 'common.yesterday': 'Вчера', 'common.week': 'Неделя',
    'common.month': 'Месяц', 'common.custom': 'Период', 'common.save': 'Сохранить', 'common.cancel': 'Отмена',
    'common.logout': 'Выйти', 'common.theme.dark': 'Тёмная тема', 'common.theme.light': 'Светлая тема',
    'role.admin': 'Администратор', 'role.operator': 'Оператор',
    'dash.live': 'Онлайн', 'dash.updating': 'Обновление', 'dash.connecting': 'Подключение…',
    'dash.kpi.total': 'Звонки', 'dash.kpi.in': 'Входящие', 'dash.kpi.out': 'Исходящие',
    'dash.kpi.avg': 'Среднее время', 'dash.kpi.active': 'Активные операторы',
    'dash.hourly': 'Активность по часам', 'dash.operators': 'Операторы', 'dash.recent': 'Последние звонки',
    'dash.incoming': 'Входящие', 'dash.outgoing': 'Исходящие',
    'st.total': 'Всего', 'st.answered': 'Ответили', 'st.missed': 'Пропущенные',
    'st.success': 'Успешные', 'st.failed': 'Не дозвонились', 'st.avgTalk': 'Среднее время разговора',
    'st.totalTalk': 'Общее время разговора', 'st.surveys': 'Анкет заполнено', 'st.opTable': 'Статистика операторов',
    'st.name': 'Имя', 'st.online': 'Онлайн', 'st.inCalls': 'Вход.', 'st.outCalls': 'Исход.', 'st.time': 'Время',
    'tv.title': 'Мониторинг операторов', 'tv.onLine': 'На линии', 'tv.offLine': 'Не на линии',
    'tv.talking': 'Разговаривает', 'tv.dnd': 'Пилот (DND)', 'tv.ringing': 'Звонок', 'tv.status': 'Статус',
    'tv.unfilled': 'Незаполненные анкеты', 'tv.servers': 'Серверы', 'tv.missed': 'Пропущенные', 'tv.unknown': 'Неизвестно',
  },
  en: {
    'app.subtitle': 'Call Center',
    'nav.dashboard': 'Dashboard', 'nav.survey': 'Survey', 'nav.calls': 'Calls', 'nav.surveyReport': 'Survey report',
    'nav.servers': 'Servers', 'nav.staff': 'Staff', 'nav.sessions': 'Sessions', 'nav.analytics': 'Analytics', 'nav.feedback': 'Client reviews', 'nav.auditLog': 'Audit log', 'nav.tv': 'TV screen',
    'common.all': 'All', 'common.today': 'Today', 'common.yesterday': 'Yesterday', 'common.week': 'Week',
    'common.month': 'Month', 'common.custom': 'Custom', 'common.save': 'Save', 'common.cancel': 'Cancel',
    'common.logout': 'Log out', 'common.theme.dark': 'Dark mode', 'common.theme.light': 'Light mode',
    'role.admin': 'Administrator', 'role.operator': 'Operator',
    'dash.live': 'Live', 'dash.updating': 'Updating', 'dash.connecting': 'Connecting…',
    'dash.kpi.total': 'Calls', 'dash.kpi.in': 'Incoming', 'dash.kpi.out': 'Outgoing',
    'dash.kpi.avg': 'Avg talk', 'dash.kpi.active': 'Active operators',
    'dash.hourly': 'Hourly activity', 'dash.operators': 'Operators', 'dash.recent': 'Recent calls',
    'dash.incoming': 'Incoming', 'dash.outgoing': 'Outgoing',
    'st.total': 'Total', 'st.answered': 'Answered', 'st.missed': 'Missed',
    'st.success': 'Successful', 'st.failed': 'Not reached', 'st.avgTalk': 'Avg talk time',
    'st.totalTalk': 'Total talk time', 'st.surveys': 'Surveys filled', 'st.opTable': 'Operator statistics',
    'st.name': 'Name', 'st.online': 'Online', 'st.inCalls': 'In', 'st.outCalls': 'Out', 'st.time': 'Time',
    'tv.title': 'Operators monitoring', 'tv.onLine': 'On line', 'tv.offLine': 'Off line',
    'tv.talking': 'Talking', 'tv.dnd': 'Pilot (DND)', 'tv.ringing': 'Ringing', 'tv.status': 'Status',
    'tv.unfilled': 'Unfilled surveys', 'tv.servers': 'Servers', 'tv.missed': 'Missed', 'tv.unknown': 'Unknown',
  },
}

// reaktiv tarjima: locale.value o'qiladi → o'zgarsa qayta render bo'ladi
export function t(key) {
  const dict = M[locale.value] || M.uz
  return dict[key] || M.uz[key] || key
}
