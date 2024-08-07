from locust import HttpUser, task
from faker import Faker
import string
import random

fake = Faker()

class HelloWorldUser(HttpUser):
    host = "http://localhost:3000"
    token = None
    
    @task
    def create_user(self):
        username = fake.user_name() + fake.user_name() + fake.password(length=5)
        password = fake.password(length=10)
        email_prefix = ''.join(random.choices(string.ascii_letters + string.digits, k=10))
        email_domain = fake.domain_name()
        email = f"{email_prefix}@{email_domain}"
        
        response = self.client.post("/v1/users", json={"username": username, "email": email, "password": password})
        
            
        self.user_data = {"email": email, "password": password}

        if response.status_code != 200:
            print(response.text)

    @task
    def login(self):
        if hasattr(self, 'user_data'):
            email = self.user_data['email']
            password = self.user_data['password']
            
            response = self.client.post("/v1/sessions", json={"email": email, "password": password})
            
            if response.status_code == 201:
                self.token = response.json().get('token')
            else:
                print("Login failed")

    @task
    def create_talks(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.post("/v1/talk", json={"type": fake.word(), "time": fake.random_number(digits=2).__str__()},headers=headers)
            
            if response.status_code != 201:
                print("Failed to access /v1/talk")
                print(response.json().get('error'))

    @task
    def get_talks(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.get("/v1/talk", headers=headers)
            
            if response.status_code != 200:
                print("Failed to access /v1/talk")
                print(response.json().get('error'))

    @task
    def create_media(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.post("/v1/medias", json={"url":"https://www.youtube.com/watch?v=NuS39O7_6ps", "type":"Youtube", "watch_type":"passive", "target_language": "pt"},headers=headers)
            
            if response.status_code != 201:
                print("Failed to access /v1/talk")
                print(response.json().get('error'))


    @task
    def get_media(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.get("/v1/medias", headers=headers)
            
            if response.status_code != 200:
                print("Failed to access /v1/medias")
                print(response.json().get('error'))

    @task
    def create_anki(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.post("/v1/anki", json={"reviewed":fake.random_number(digits=2), "newCards":fake.random_number(digits=2), "time":fake.random_number(digits=2), "target_language": "en"},headers=headers)

            if response.status_code != 201:
                print("Failed to access /v1/talk")
                print(response.json().get('error'))


    @task
    def get_anki(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.get("/v1/anki", headers=headers)

            if response.status_code != 200:
                print("Failed to access /v1/anki")
                print(response.json().get('error'))
                
    @task
    def create_vocabulary(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.post("/v1/vocabulary", json={"reviewed":fake.random_number(digits=4), "target_language": "en"},headers=headers)

            if response.status_code != 201:
                print("Failed to access /v1/vocabulary")
                print(response.json().get('error'))


    @task
    def get_vocabulary(self):
        if self.token:
            headers = {'Authorization': f'Bearer {self.token}'}
            response = self.client.get("/v1/vocabulary", headers=headers)

            if response.status_code != 200:
                print("Failed to access /v1/vocabulary")
                print(response.json().get('error'))
