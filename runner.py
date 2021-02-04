import matplotlib.pyplot as plt
import os
import numpy as np
import time


class Parser:
    def __init__(self, filepath="output.txt") -> None:
        self.filepath = os.path.join(
            os.path.dirname(__file__), 'client', filepath)

    def check(self):
        if not os.path.exists(self.filepath):
            raise Exception(f"File {self.filepath} does not exits.")

    def openigFile(self) -> list:
        self.check()
        file = open(self.filepath)
        lines = file.readlines()
        file.close()
        return lines

    def execute(self):
        lines = self.openigFile()
        lines = lines[1:]
        windowSize, times = str(lines[-1]).split(',')

        print('I*******************I')
        print(times[:-2])
        print('I*******************I')
        return int(windowSize), float(times[:-2])


class Drawer:
    def __init__(self) -> None:
        self.XDATA = []
        self.YDATA = []

    def manageData(self):
        self.XDATA = np.array([self.XDATA])
        self.YDATA = np.array([self.YDATA])
        return np.hstack([np.reshape(self.XDATA, (-1, 1)), np.reshape(self.YDATA, (-1, 1))])

    def drawingBytes(self):
        DATA = self.manageData()
        DATA = DATA[DATA[:, 0].argsort()]
        fig2 = plt.figure(2, figsize=(10, 8))
        plt.plot(DATA[:, 0], DATA[:, 1], 'k-')
        plt.scatter(DATA[:, 0], DATA[:, 1], c='r',
                    marker='o', s=100, alpha=0.6, zorder=100)
        plt.bar(DATA[:, 0], DATA[:, 1], color="orange", width=2,
                linewidth=1.5, edgecolor=np.random.random((255, 3)))
        plt.xlabel(r'$Frame$ $Bytes$')
        plt.ylabel(r'$Time$ $In$ $Milliseconds$')
        # plt.xticks(DATA[:, 0])
        print(np.argmin(DATA[:, 0]))
        plt.axis([DATA[np.argmin(DATA[:, 0]), 0] - 10, DATA[np.argmax(DATA[:, 0]), 0] +
                  10, DATA[np.argmin(DATA[:, 1]), 1] - 5, DATA[np.argmax(DATA[:, 1]), 1] + 5])
        # plt.yticks(DATA[:, 1])
        # plt.yscale()
        plt.grid(True)
        fig2.savefig("DifferentBytes.png")
        fig2.show()

    def setDataForBytes(self):
        import glob
        parser = Parser()
        files = glob.glob(os.path.join(os.path.abspath(
            os.path.dirname(__file__)), 'output_different_bytes', '*.txt'), recursive=True)
        for file in files:
            parser.filepath = file
            b, times = parser.execute()
            self.XDATA.append(b)
            self.YDATA.append(times)

    def drawWindowSizes(self):
        DATA = self.manageData()
        fig1 = plt.figure(1, figsize=(10, 8))
        plt.plot(DATA[:, 0], DATA[:, 1], 'r-')
        plt.scatter(DATA[:, 0], DATA[:, 1], c='b',
                    marker="o", s=100, zorder=100)
        plt.bar(DATA[:, 0], DATA[:, 1], width=0.9, color="orange")
        plt.xticks(DATA[:, 0])
        plt.yticks(DATA[:, 1])
        plt.xlabel(r"$Window Sizes$")
        plt.ylabel(r"$Times$ $In$ $milliseconds$")
        fig1.savefig("windows.png")
        plt.grid(True)
        fig1.show()

    def execute(self):
        self.drawWindowSizes()
        self.XDATA, self.YDATA = [], []
        # self.setDataForBytes()
        # print(self.XDATA, self.YDATA)
        self.drawingBytes()
        plt.show()


class Executor:
    def __init__(self, windowSizes=[8, 32, 64, 128]) -> None:
        self.windowSizes = windowSizes
        self.serverPath = os.path.join(os.path.abspath(
            os.path.dirname(__file__)), 'server', 'server.go')
        self.clientPath = os.path.join(os.path.abspath(
            os.path.dirname(__file__)), 'client', 'client.go')
        print(self.serverPath)
        self.parser = Parser()
        self.drawer = Drawer()

    def executor(self, folderpath, filepath, windowSize=8):
        import subprocess
        proc = subprocess.Popen(
            f"cd {folderpath} && go run {filepath} {windowSize}", shell=True)
        return proc

    def execute(self):
        proc, proc1 = None, None
        try:
            for size in self.windowSizes:
                print(f"starting the window size: {size}")
                proc = self.executor(os.path.dirname(
                    self.serverPath), self.serverPath, size)
                time.sleep(.5)
                proc1 = self.executor(os.path.dirname(
                    self.clientPath), self.clientPath, size)
                proc1.wait()
                proc.wait()
                print(f"end the window size: {size}")
                windowsize, times = self.parser.execute()
                self.drawer.XDATA.append(windowsize)
                self.drawer.YDATA.append(times)
        except:
            proc.kill()
            proc1.kill()

        self.drawer.execute()


if __name__ == '__main__':
    Executor().execute()
