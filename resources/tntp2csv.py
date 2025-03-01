import pandas as pd

CONVERTU = 0.3048

file_path = 'resources/Barcelona_net.tntp'

with open(file_path, 'r') as file:
    tntp_content = file.readlines()


header_line = tntp_content[8]  
data_lines = tntp_content[9:]  


header = header_line.replace('~', '').replace(';', '').strip().split()
data = [line.replace(';', '').strip().split() for line in data_lines if line.strip()]


df = pd.DataFrame(data, columns=header)
df = df[['init_node', 'term_node', 'length', 'speed']]

df['length'] = df['length'].astype(float) * CONVERTU

df['speed'] = df['speed'].astype(float) * CONVERTU / 60

df['speed'] = 5.0

csv_file_path = 'resources/Barcelona_net.csv'
df.to_csv(csv_file_path, index=False)

