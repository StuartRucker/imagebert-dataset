
import os
import pandas as pd
import numpy as np

import cv2
import matplotlib.pyplot as plt
import random



def generate():
    # select a random file from data/csv/
    all_files = os.listdir(os.path.join(os.getcwd(), 'data', 'csv'))


    chosen_file = random.choice(all_files)
    # extract the id from the filename formated as {id}-tokens.csv
    id = chosen_file.split('-')[0]
    # id = id[1:-1]




    df = pd.read_csv(os.path.join(os.getcwd(), 'data', 'csv', chosen_file))
    print(df.columns)

    # select a random row

    # make folder for demos
    foldername = ""
    if not os.path.exists(os.path.join(foldername, 'demos')):
        os.mkdir(os.path.join(foldername, 'demos'))

    row = df.sample(n=1)

    image_path = os.path.join("data/img", f"{id}-{row.iloc[0]['NODEID']}.png")

    # read the image in
    img = cv2.imread(image_path)

    # for every token with this NodeID, draw a box around it
    for index, tmp_row in df[df.NODEID == row.iloc[0]['NODEID']].iterrows():
        xmin, ymin = int(tmp_row['X']), int(tmp_row['Y'])
        xmax, ymax = xmin + int(tmp_row['WIDTH']), ymin + int(tmp_row['HEIGHT'])
        # draw a box around the token
        cv2.rectangle(img, (xmin, ymin), (xmax, ymax), (255, 0, 0), 1)

    print(row.iloc[0]['TOKEN'])
    print(row.iloc[0])


    xmin, ymin = int(row.iloc[0]['X']), int(row.iloc[0]['Y'])
    xmax, ymax = xmin + int(row.iloc[0]['WIDTH']), ymin + int(row.iloc[0]['HEIGHT'])
    # draw a box around the token
    # cv2.rectangle(img, (xmin, ymin), (xmax, ymax), (255, 0, 0), -1)
    # write the token text
    # cv2.putText(img, row.iloc[0]['TOKEN'], (50, 50), cv2.FONT_HERSHEY_SIMPLEX, 1, (0, 0, 255), 2)

    # save the image to demos/{random_number}.png
    cv2.imwrite(os.path.join(foldername, f'demos/{random.random()}.png'), img)


for i in range(30):
    try:
        generate()
    except:
        print("error")