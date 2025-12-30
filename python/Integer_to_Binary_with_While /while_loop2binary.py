# while_loop2binary.py


def conv2binary():
    
    n = int(input("Enter a non-negative integer: "))
    if n < 0:
        print("Please enter a non-negative integer.")
        return

    binary = ""
    if n == 0:
        binary = "0"
    else:
        while n > 0:
            remainder = n % 2
            binary = str(remainder) + binary
            n = n // 2
            print(type(binary))
            print(binary)
            

    print("Binary representation:", binary)


conv2binary()