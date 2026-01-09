from playwright.sync_api import sync_playwright
import json
import os

OUTPUT_DIR = 'F:/Code/StuHelper/scripts/explore_output'
os.makedirs(OUTPUT_DIR, exist_ok=True)

def save_json(filename, data):
    with open(f'{OUTPUT_DIR}/{filename}', 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

def save_screenshot(page, name):
    page.screenshot(path=f'{OUTPUT_DIR}/{name}.png', full_page=True)

def save_html(page, name):
    with open(f'{OUTPUT_DIR}/{name}.html', 'w', encoding='utf-8') as f:
        f.write(page.content())

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page()

    # 收集 API 响应
    api_data = []

    def handle_response(response):
        if 'api' in response.url.lower():
            try:
                ct = response.headers.get('content-type', '')
                body = response.json() if 'application/json' in ct else None
                api_data.append({
                    'url': response.url,
                    'status': response.status,
                    'body': body
                })
            except:
                pass

    page.on('response', handle_response)

    # 要访问的页面列表
    pages_to_visit = [
        ('https://courses.pinzhixiaoyuan.com/', '01_home'),
        ('https://courses.pinzhixiaoyuan.com/courses/list', '02_courses_list'),
        ('https://courses.pinzhixiaoyuan.com/courses/view/1033', '03_course_detail'),
        ('https://courses.pinzhixiaoyuan.com/reviews/latest', '04_reviews_latest'),
        ('https://courses.pinzhixiaoyuan.com/search', '05_search'),
        ('https://courses.pinzhixiaoyuan.com/reviews/post', '06_post_review'),
        ('https://courses.pinzhixiaoyuan.com/about', '07_about'),
        ('https://courses.pinzhixiaoyuan.com/faq', '08_faq'),
    ]

    for i, (url, name) in enumerate(pages_to_visit):
        print(f"{i+1}. 访问 {url}...")
        page.goto(url)
        page.wait_for_load_state('networkidle')
        page.wait_for_timeout(1500)
        save_screenshot(page, name)
        save_html(page, name)

    # 保存 API 数据
    save_json('api_responses.json', api_data)
    print(f"\n收集到 {len(api_data)} 个 API 响应")
    print("探索完成！截图和数据已保存到:", OUTPUT_DIR)
    browser.close()
